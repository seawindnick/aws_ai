package repository

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors"
	s3vdoc "github.com/aws/aws-sdk-go-v2/service/s3vectors/document"
	s3vtypes "github.com/aws/aws-sdk-go-v2/service/s3vectors/types"
)

type S3VectorsRepo struct {
	client     *s3vectors.Client
	bucketName string
}

func NewS3VectorsRepo(client *s3vectors.Client, bucketName string) *S3VectorsRepo {
	return &S3VectorsRepo{client: client, bucketName: bucketName}
}

func (r *S3VectorsRepo) Put(ctx context.Context, questionID, userID, subject string, vector []float64) error {
	f32 := toFloat32(vector)
	meta := s3vdoc.NewLazyDocument(map[string]any{
		"user_id": userID,
		"subject": subject,
	})

	_, err := r.client.PutVectors(ctx, &s3vectors.PutVectorsInput{
		VectorBucketName: aws.String(r.bucketName),
		Vectors: []s3vtypes.PutInputVector{
			{
				Key:      aws.String(questionID),
				Data:     &s3vtypes.VectorDataMemberFloat32{Value: f32},
				Metadata: meta,
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
	f32 := toFloat32(vector)

	// Build metadata filter using document.NewLazyDocument
	var filterMap map[string]any
	if subject != "" {
		filterMap = map[string]any{
			"$and": []any{
				map[string]any{"user_id": map[string]any{"$eq": userID}},
				map[string]any{"subject": map[string]any{"$eq": subject}},
			},
		}
	} else {
		filterMap = map[string]any{
			"user_id": map[string]any{"$eq": userID},
		}
	}

	out, err := r.client.QueryVectors(ctx, &s3vectors.QueryVectorsInput{
		VectorBucketName: aws.String(r.bucketName),
		QueryVector:      &s3vtypes.VectorDataMemberFloat32{Value: f32},
		TopK:             aws.Int32(int32(k)),
		Filter:           s3vdoc.NewLazyDocument(filterMap),
		ReturnDistance:   true,
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

func toFloat32(v []float64) []float32 {
	f32 := make([]float32, len(v))
	for i, x := range v {
		f32[i] = float32(x)
	}
	return f32
}
