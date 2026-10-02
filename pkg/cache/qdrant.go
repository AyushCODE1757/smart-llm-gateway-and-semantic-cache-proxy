package cache

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

func NewQDrantClient(host string) (*qdrant.Client, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   6334,
		APIKey: "",
		UseTLS: false,
	})

	if err != nil {
		return nil, err
	}

	return client, nil
}

type SimilaritySearchResult struct {
	Value string
	Score float32
}

type QdrantCache struct {
	client         *qdrant.Client
	collectionName string
	threshold      float32
}

func (q *QdrantCache) SimilaritySearch(ctx context.Context, embedding []float32) (*SimilaritySearchResult, error) {
	queryResult, err := q.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: q.collectionName,
		Query:          qdrant.NewQueryDense(embedding),
		Limit:          qdrant.PtrOf(uint64(1)),
		WithPayload:    qdrant.NewWithPayload(true),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant similarity search failed: %w", err)
	}
	if len(queryResult) == 0 {
		return nil, nil
	}
	hit := queryResult[0]
	if hit.Score < q.threshold {
		return nil, nil
	}
	responsePayload, ok := hit.Payload["response"]
	if !ok {
		return nil, fmt.Errorf("qdrant result missing response payload")
	}

	responseValue, ok := responsePayload.GetKind().(*qdrant.Value_StringValue)
	if !ok {
		return nil, fmt.Errorf("qdrant response payload is not a string")

	}
	return &SimilaritySearchResult{
		Value: responseValue.StringValue,
		Score: hit.Score,
	}, nil
}

func (q *QdrantCache) AddTestPoint(
	ctx context.Context,
	id uint64,
	embedding []float32,
	response string,
) error {
	_, err := q.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: q.collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id:      qdrant.NewIDNum(id),
				Vectors: qdrant.NewVectorsDense(embedding),
				Payload: map[string]*qdrant.Value{
					"response": qdrant.NewValueString(response),
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to insert test point: %w", err)
	}

	return nil
}
