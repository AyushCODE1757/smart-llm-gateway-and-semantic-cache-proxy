package cache

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

func NewQDrantClient(host string, port int) (*qdrant.Client, error) {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
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

func NewQdrantCache(
	host string,
	port int,
	collectionName string,
	threshold float32,
) (*QdrantCache, error) {
	client, err := NewQDrantClient(host, port)
	if err != nil {
		return nil, fmt.Errorf("failed to create Qdrant client: %w", err)
	}

	return &QdrantCache{
		client:         client,
		collectionName: collectionName,
		threshold:      threshold,
	}, nil
}

func (q *QdrantCache) InitCollection(ctx context.Context) error {
	exists, err := q.client.CollectionExists(ctx, q.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if exists {
		fmt.Printf("Collection '%s' already exists.\n", q.collectionName)
		return nil
	}

	err = q.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: q.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(
			&qdrant.VectorParams{
				Size:     768,
				Distance: qdrant.Distance_Cosine,
			},
		),
	})
	if err != nil {
		return fmt.Errorf(
			"failed to create collection %s: %w",
			q.collectionName,
			err,
		)
	}

	fmt.Printf(
		"Collection %s created successfully!\n",
		q.collectionName,
	)

	return nil
}

func (q *QdrantCache) Close() error {
	return q.client.Close()
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

func (q *QdrantCache) AddPoint(
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
		return fmt.Errorf("failed to insert cache point: %w", err)
	}

	return nil
}
