package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors"
	s3vtypes "github.com/aws/aws-sdk-go-v2/service/s3vectors/types"
)

type S3VectorsRepo struct {
	client     *s3vectors.Client
	bucketName string
}

func NewS3VectorsRepo(client *s3vectors.Client, bucketName string) *S3VectorsRepo {
	return &S3VectorsRepo{client: client, bucketName: bucketName}
}

type VectorMetadata struct {
	UserID  string `json:"user_id"`
	Subject string `json:"subject"`
}

func (r *S3VectorsRepo) Put(ctx context.Context, questionID, userID, subject string, vector []float64) error {
	meta, err := json.Marshal(VectorMetadata{UserID: userID, Subject: subject})
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	f32 := make([]float32, len(vector))
	for i, v := range vector {
		f32[i] = float32(v)
	}

	_, err = r.client.PutVectors(ctx, &s3vectors.PutVectorsInput{
		VectorBucketName: aws.String(r.bucketName),
		Vectors: []s3vtypes.PutInputVector{
			{
				Key:      aws.String(questionID),
				Data:     &s3vtypes.VectorDataMemberFloat32{Value: f32},
				Metadata: &s3vtypes.DocumentMemberString{Value: string(meta)},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("put vector: %w", err)
	}
	return nil
}

type VectorResult struct {
	QuestionID string
	Score      float64
}

func (r *S3VectorsRepo) Query(ctx context.Context, userID, subject string, vector []float64, k int) ([]*VectorResult, error) {
	f32 := make([]float32, len(vector))
	for i, v := range vector {
		f32[i] = float32(v)
	}

	// Build metadata filter: user_id must match; optionally filter subject
	filterExpr := fmt.Sprintf(`{"user_id": {"$eq": %q}}`, userID)
	if subject != "" {
		filterExpr = fmt.Sprintf(`{"$and": [{"user_id": {"$eq": %q}}, {"subject": {"$eq": %q}}]}`, userID, subject)
	}

	out, err := r.client.QueryVectors(ctx, &s3vectors.QueryVectorsInput{
		VectorBucketName: aws.String(r.bucketName),
		QueryVector:      &s3vtypes.VectorDataMemberFloat32{Value: f32},
		TopK:             aws.Int32(int32(k)),
		Filter:           &s3vtypes.DocumentMemberString{Value: filterExpr},
		ReturnDistance:   aws.Bool(true),
	})
	if err != nil {
		return nil, fmt.Errorf("query vectors: %w", err)
	}

	results := make([]*VectorResult, 0, len(out.Vectors))
	for _, v := range out.Vectors {
		score := 0.0
		if v.Distance != nil {
			// S3 Vectors returns cosine distance; convert to similarity
			score = 1.0 - float64(*v.Distance)
			if score < 0 {
				score = 0
			}
		}
		results = append(results, &VectorResult{
			QuestionID: aws.ToString(v.Key),
			Score:      score,
		})
	}
	return results, nil
}

func (r *S3VectorsRepo) Delete(ctx context.Context, questionID string) error {
	_, err := r.client.DeleteVectors(ctx, &s3vectors.DeleteVectorsInput{
		VectorBucketName: aws.String(r.bucketName),
		Keys:             []string{questionID},
	})
	if err != nil {
		return fmt.Errorf("delete vector: %w", err)
	}
	return nil
}
