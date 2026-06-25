import * as cdk from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as dynamodb from 'aws-cdk-lib/aws-dynamodb';
import * as cognito from 'aws-cdk-lib/aws-cognito';
import * as lambda from 'aws-cdk-lib/aws-lambda';
import * as lambdaEventSources from 'aws-cdk-lib/aws-lambda-event-sources';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as apigatewayv2 from 'aws-cdk-lib/aws-apigatewayv2';
import * as apigatewayv2Integrations from 'aws-cdk-lib/aws-apigatewayv2-integrations';
import * as apigatewayv2Authorizers from 'aws-cdk-lib/aws-apigatewayv2-authorizers';

export interface WrongQuestionStackProps extends cdk.StackProps {
  envName?: string;
}

export class WrongQuestionStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: WrongQuestionStackProps) {
    super(scope, id, props);

    const envName = props?.envName ?? 'prod';

    // ─── Cognito User Pool ────────────────────────────────────────────────────
    const userPool = new cognito.UserPool(this, 'UserPool', {
      userPoolName: `wrong-question-${envName}`,
      selfSignUpEnabled: true,
      signInAliases: { email: true },
      standardAttributes: {
        email: { required: true, mutable: true },
      },
      passwordPolicy: {
        minLength: 8,
        requireLowercase: true,
        requireDigits: true,
        requireUppercase: false,
        requireSymbols: false,
      },
      accountRecovery: cognito.AccountRecovery.EMAIL_ONLY,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    const userPoolClient = new cognito.UserPoolClient(this, 'UserPoolClient', {
      userPool,
      authFlows: {
        userPassword: true,
        userSrp: true,
      },
      generateSecret: false,
    });

    // ─── DynamoDB: users ──────────────────────────────────────────────────────
    const usersTable = new dynamodb.Table(this, 'Users', {
      tableName: `wrong-question-users-${envName}`,
      partitionKey: { name: 'user_id', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // GSI: email_index — for login lookup
    usersTable.addGlobalSecondaryIndex({
      indexName: 'email_index',
      partitionKey: { name: 'email', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });

    // ─── DynamoDB: questions ──────────────────────────────────────────────────
    const questionsTable = new dynamodb.Table(this, 'Questions', {
      tableName: `wrong-question-questions-${envName}`,
      partitionKey: { name: 'question_id', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      stream: dynamodb.StreamViewType.NEW_IMAGE,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // GSI: user_created_index — for listing a user's questions
    questionsTable.addGlobalSecondaryIndex({
      indexName: 'user_created_index',
      partitionKey: { name: 'user_id', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'created_at', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });

    // ─── DynamoDB: review_schedules ───────────────────────────────────────────
    const reviewSchedulesTable = new dynamodb.Table(this, 'ReviewSchedules', {
      tableName: `wrong-question-review-schedules-${envName}`,
      partitionKey: { name: 'user_id', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'question_id', type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
      timeToLiveAttribute: 'ttl',
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    reviewSchedulesTable.addGlobalSecondaryIndex({
      indexName: 'user_date_index',
      partitionKey: { name: 'user_id', type: dynamodb.AttributeType.STRING },
      sortKey: { name: 'next_review_at', type: dynamodb.AttributeType.STRING },
      projectionType: dynamodb.ProjectionType.ALL,
    });

    // ─── S3: images bucket ────────────────────────────────────────────────────
    // Structure: images/{user_id}/{question_id}.jpg
    const imagesBucket = new s3.Bucket(this, 'ImagesBucket', {
      bucketName: `wrong-question-images-${this.account}-${envName}`,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      cors: [
        {
          allowedMethods: [s3.HttpMethods.PUT, s3.HttpMethods.GET],
          allowedOrigins: ['*'],
          allowedHeaders: ['*'],
          maxAge: 3000,
        },
      ],
      removalPolicy: cdk.RemovalPolicy.DESTROY,
      autoDeleteObjects: true,
    });

    // ─── Lambda: image-analyzer (triggered by S3 PUT to images/) ─────────────
    const analyzerRole = new iam.Role(this, 'AnalyzerRole', {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaBasicExecutionRole'),
      ],
    });

    analyzerRole.addToPolicy(new iam.PolicyStatement({
      actions: ['s3:GetObject'],
      resources: [`${imagesBucket.bucketArn}/images/*`],
    }));

    analyzerRole.addToPolicy(new iam.PolicyStatement({
      actions: ['bedrock:InvokeModel'],
      resources: ['*'],
    }));

    analyzerRole.addToPolicy(new iam.PolicyStatement({
      actions: ['dynamodb:PutItem', 'dynamodb:UpdateItem'],
      resources: [questionsTable.tableArn],
    }));

    const analyzerLogGroup = new logs.LogGroup(this, 'AnalyzerLogGroup', {
      logGroupName: `/aws/lambda/wrong-question-image-analyzer-${envName}`,
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    const imageAnalyzer = new lambda.Function(this, 'ImageAnalyzer', {
      functionName: `wrong-question-image-analyzer-${envName}`,
      runtime: lambda.Runtime.PROVIDED_AL2023,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../lambda/image-analyzer/dist'),
      role: analyzerRole,
      timeout: cdk.Duration.minutes(3),
      memorySize: 512,
      logGroup: analyzerLogGroup,
      environment: {
        DYNAMO_TABLE_QUESTIONS: questionsTable.tableName,
        VISION_MODEL_ID: 'us.anthropic.claude-haiku-4-5-20251001-v1:0',
      },
    });

    // Trigger on any PUT under images/ prefix
    imageAnalyzer.addEventSource(new lambdaEventSources.S3EventSource(imagesBucket, {
      events: [s3.EventType.OBJECT_CREATED],
      filters: [{ prefix: 'images/' }],
    }));

    // ─── Lambda: embedding worker (DynamoDB Streams trigger) ──────────────────
    const embeddingWorkerRole = new iam.Role(this, 'EmbeddingWorkerRole', {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaBasicExecutionRole'),
      ],
    });

    embeddingWorkerRole.addToPolicy(new iam.PolicyStatement({
      actions: ['dynamodb:UpdateItem'],
      resources: [questionsTable.tableArn],
    }));

    embeddingWorkerRole.addToPolicy(new iam.PolicyStatement({
      actions: [
        'dynamodb:GetRecords', 'dynamodb:GetShardIterator',
        'dynamodb:DescribeStream', 'dynamodb:ListStreams',
      ],
      resources: [questionsTable.tableStreamArn!],
    }));

    embeddingWorkerRole.addToPolicy(new iam.PolicyStatement({
      actions: ['bedrock:InvokeModel'],
      resources: ['*'],
    }));

    embeddingWorkerRole.addToPolicy(new iam.PolicyStatement({
      actions: ['s3vectors:PutVectors'],
      resources: ['*'],
    }));

    const embeddingWorkerLogGroup = new logs.LogGroup(this, 'EmbeddingWorkerLogGroup', {
      logGroupName: `/aws/lambda/wrong-question-embedding-worker-${envName}`,
      retention: logs.RetentionDays.ONE_WEEK,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    const embeddingWorker = new lambda.Function(this, 'EmbeddingWorker', {
      functionName: `wrong-question-embedding-worker-${envName}`,
      runtime: lambda.Runtime.PROVIDED_AL2023,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../lambda/embedding-worker/dist'),
      role: embeddingWorkerRole,
      timeout: cdk.Duration.minutes(5),
      memorySize: 512,
      logGroup: embeddingWorkerLogGroup,
      environment: {
        DYNAMO_TABLE_QUESTIONS: questionsTable.tableName,
        EMBEDDING_MODEL_ID: 'amazon.titan-embed-text-v2:0',
      },
    });

    // Trigger on INSERT events (status == "done") from questions table stream
    embeddingWorker.addEventSource(new lambdaEventSources.DynamoEventSource(questionsTable, {
      startingPosition: lambda.StartingPosition.LATEST,
      batchSize: 10,
      bisectBatchOnError: true,
      retryAttempts: 2,
      filters: [
        lambda.FilterCriteria.filter({
          eventName: lambda.FilterRule.isEqual('MODIFY'),
          dynamodb: {
            NewImage: {
              status: { S: lambda.FilterRule.isEqual('done') },
            },
          },
        }),
      ],
    }));

    // ─── Lambda: API handler ──────────────────────────────────────────────────
    const apiRole = new iam.Role(this, 'ApiRole', {
      assumedBy: new iam.ServicePrincipal('lambda.amazonaws.com'),
      managedPolicies: [
        iam.ManagedPolicy.fromAwsManagedPolicyName('service-role/AWSLambdaBasicExecutionRole'),
      ],
    });

    apiRole.addToPolicy(new iam.PolicyStatement({
      actions: [
        'dynamodb:GetItem', 'dynamodb:PutItem', 'dynamodb:UpdateItem',
        'dynamodb:DeleteItem', 'dynamodb:Query', 'dynamodb:Scan',
      ],
      resources: [
        usersTable.tableArn,
        `${usersTable.tableArn}/index/*`,
        questionsTable.tableArn,
        `${questionsTable.tableArn}/index/*`,
        reviewSchedulesTable.tableArn,
        `${reviewSchedulesTable.tableArn}/index/*`,
      ],
    }));

    apiRole.addToPolicy(new iam.PolicyStatement({
      actions: ['s3:PutObject', 's3:GetObject', 's3:DeleteObject'],
      resources: [`${imagesBucket.bucketArn}/images/*`],
    }));

    apiRole.addToPolicy(new iam.PolicyStatement({
      actions: ['s3:GeneratePresignedUrl'],
      resources: [`${imagesBucket.bucketArn}/*`],
      // GetObject pre-sign is covered by s3:GetObject; PutObject pre-sign by s3:PutObject
    }));

    apiRole.addToPolicy(new iam.PolicyStatement({
      actions: ['bedrock:InvokeModel'],
      resources: ['*'],
    }));

    apiRole.addToPolicy(new iam.PolicyStatement({
      actions: [
        'cognito-idp:AdminCreateUser', 'cognito-idp:AdminSetUserPassword',
        'cognito-idp:AdminGetUser', 'cognito-idp:ListUsers',
        'cognito-idp:InitiateAuth', 'cognito-idp:SignUp',
        'cognito-idp:ConfirmSignUp', 'cognito-idp:GlobalSignOut',
      ],
      resources: [userPool.userPoolArn],
    }));

    const apiLogGroup = new logs.LogGroup(this, 'ApiLogGroup', {
      logGroupName: `/aws/lambda/wrong-question-api-${envName}`,
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Placeholder — deploy real binary when backend Lambda is compiled
    const apiHandler = new lambda.Function(this, 'ApiHandler', {
      functionName: `wrong-question-api-${envName}`,
      runtime: lambda.Runtime.PROVIDED_AL2023,
      handler: 'bootstrap',
      code: lambda.Code.fromAsset('../lambda/api-handler/dist'),
      role: apiRole,
      timeout: cdk.Duration.seconds(30),
      memorySize: 256,
      logGroup: apiLogGroup,
      environment: {
        AWS_REGION_NAME: this.region,
        DYNAMO_TABLE_USERS: usersTable.tableName,
        DYNAMO_TABLE_QUESTIONS: questionsTable.tableName,
        DYNAMO_TABLE_REVIEW_SCHEDULES: reviewSchedulesTable.tableName,
        IMAGES_BUCKET: imagesBucket.bucketName,
        COGNITO_USER_POOL_ID: userPool.userPoolId,
        COGNITO_CLIENT_ID: userPoolClient.userPoolClientId,
      },
    });

    // ─── API Gateway HTTP API ─────────────────────────────────────────────────
    const httpApi = new apigatewayv2.HttpApi(this, 'HttpApi', {
      apiName: `wrong-question-api-${envName}`,
      corsPreflight: {
        allowHeaders: ['Content-Type', 'Authorization'],
        allowMethods: [apigatewayv2.CorsHttpMethod.ANY],
        allowOrigins: ['*'],
      },
    });

    // Cognito JWT authorizer
    const authorizer = new apigatewayv2Authorizers.HttpJwtAuthorizer('CognitoAuth', userPool.userPoolProviderUrl, {
      jwtAudience: [userPoolClient.userPoolClientId],
    });

    const lambdaIntegration = new apigatewayv2Integrations.HttpLambdaIntegration('ApiIntegration', apiHandler);

    // Public routes (no auth)
    httpApi.addRoutes({
      path: '/api/health',
      methods: [apigatewayv2.HttpMethod.GET],
      integration: lambdaIntegration,
    });

    httpApi.addRoutes({
      path: '/api/auth/{proxy+}',
      methods: [apigatewayv2.HttpMethod.ANY],
      integration: lambdaIntegration,
    });

    // Protected routes
    httpApi.addRoutes({
      path: '/api/{proxy+}',
      methods: [apigatewayv2.HttpMethod.ANY],
      integration: lambdaIntegration,
      authorizer,
    });

    // ─── Outputs ──────────────────────────────────────────────────────────────
    new cdk.CfnOutput(this, 'ApiUrl', {
      value: httpApi.apiEndpoint,
      description: 'API Gateway endpoint URL',
    });

    new cdk.CfnOutput(this, 'UserPoolId', {
      value: userPool.userPoolId,
      description: 'Cognito User Pool ID',
    });

    new cdk.CfnOutput(this, 'UserPoolClientId', {
      value: userPoolClient.userPoolClientId,
      description: 'Cognito App Client ID',
    });

    new cdk.CfnOutput(this, 'ImagesBucketName', {
      value: imagesBucket.bucketName,
      description: 'S3 bucket for question images — upload to images/{user_id}/{question_id}.jpg',
    });

    new cdk.CfnOutput(this, 'QuestionsTableName', {
      value: questionsTable.tableName,
      description: 'DynamoDB questions table',
    });
  }
}
