import * as cdk from 'aws-cdk-lib';
import { WrongQuestionStack } from './wrong-question-stack';

const app = new cdk.App();

new WrongQuestionStack(app, 'WrongQuestion', {
  envName: app.node.tryGetContext('envName') ?? 'prod',
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: process.env.CDK_DEFAULT_REGION ?? 'us-east-1',
  },
});
