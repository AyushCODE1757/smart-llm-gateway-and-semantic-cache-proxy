package cache

import (
	"context"
	"fmt"

	"github.com/qdrant/go-client/qdrant"
)

type SemanticCache struct {
	qdrantClient   *qdrant.Client
	collectionName string
}

func NewSemanticCache(host string, port int, collectionName string) *SemanticCache {
	client, err := qdrant.NewClient(&qdrant.Config{
		Host: host,
		Port: port,
	})

	if err != nil {
		fmt.Errorf("failed to create client: %w", err)

	}

	return &SemanticCache{
		qdrantClient:   client,
		collectionName: collectionName,
	}
}

func (sc *SemanticCache) InitCollection(ctx context.Context) error {
	exists, err := sc.qdrantClient.CollectionExists(ctx, sc.collectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)

	}
	if exists {
		fmt.Printf("Collection '%s' already exists.\n", sc.collectionName)
		return nil
	}

	err = sc.qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: sc.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     768,
			Distance: qdrant.Distance_Cosine,
		}),
	})
	if err != nil {
		fmt.Errorf("Failed to Create Collection %s : %w", sc.collectionName, err)

	}
	fmt.Printf("Collection %s created Successfully!!\n", sc.collectionName)
	return nil
}

func (sc *SemanticCache) Close() error {
	return sc.qdrantClient.Close()
}
