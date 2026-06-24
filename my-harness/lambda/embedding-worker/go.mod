module github.com/workshop/wrong-question/lambda/embedding-worker

go 1.25

require (
	github.com/aws/aws-lambda-go v1.47.0
	github.com/aws/aws-sdk-go-v2 v1.30.0
	github.com/aws/aws-sdk-go-v2/config v1.27.0
	github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue v1.14.0
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.10.0
	github.com/aws/aws-sdk-go-v2/service/dynamodb v1.34.0
	github.com/aws/aws-sdk-go-v2/service/s3vectors v1.0.0
	github.com/go-sql-driver/mysql v1.8.1
)
