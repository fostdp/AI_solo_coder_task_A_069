package storage

import (
	"context"
	"fmt"
	"time"

	"sim-scenario-platform/internal/config"
	"sim-scenario-platform/internal/model"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type MongoStore struct {
	client   *mongo.Client
	db       *mongo.Database
	annoCol  *mongo.Collection
	canCol   *mongo.Collection
}

func NewMongoStore(ctx context.Context, cfg *config.Config) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, fmt.Errorf("unable to connect to mongodb: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("unable to ping mongodb: %w", err)
	}
	db := client.Database(cfg.MongoDB)
	return &MongoStore{
		client:  client,
		db:      db,
		annoCol: db.Collection("annotations"),
		canCol:  db.Collection("can_logs"),
	}, nil
}

func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *MongoStore) InitCollections(ctx context.Context) error {
	_, err := s.annoCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "scene_id", Value: 1}}},
		{Keys: bson.D{{Key: "scene_id", Value: 1}, {Key: "frame_index", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create annotation indexes: %w", err)
	}
	_, err = s.canCol.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "scene_id", Value: 1}}},
	})
	if err != nil {
		return fmt.Errorf("failed to create can_logs indexes: %w", err)
	}
	return nil
}

func (s *MongoStore) CreateAnnotation(ctx context.Context, annotation *model.Annotation) error {
	if annotation.ID == "" {
		annotation.ID = bson.NewObjectID().Hex()
	}
	_, err := s.annoCol.InsertOne(ctx, annotation)
	return err
}

func (s *MongoStore) UpdateAnnotation(ctx context.Context, id string, annotation *model.Annotation) error {
	annotation.UpdatedAt = time.Now()
	_, err := s.annoCol.ReplaceOne(ctx, bson.D{{Key: "_id", Value: id}}, annotation)
	return err
}

func (s *MongoStore) GetAnnotation(ctx context.Context, id string) (*model.Annotation, error) {
	var anno model.Annotation
	err := s.annoCol.FindOne(ctx, bson.D{{Key: "_id", Value: id}}).Decode(&anno)
	if err != nil {
		return nil, err
	}
	return &anno, nil
}

func (s *MongoStore) GetAnnotationsByScene(ctx context.Context, sceneID string) ([]*model.Annotation, error) {
	cursor, err := s.annoCol.Find(ctx, bson.D{{Key: "scene_id", Value: sceneID}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var annotations []*model.Annotation
	if err := cursor.All(ctx, &annotations); err != nil {
		return nil, err
	}
	return annotations, nil
}

func (s *MongoStore) GetAnnotationsByFrame(ctx context.Context, sceneID string, frameIndex int) ([]*model.Annotation, error) {
	cursor, err := s.annoCol.Find(ctx, bson.D{
		{Key: "scene_id", Value: sceneID},
		{Key: "frame_index", Value: frameIndex},
	})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var annotations []*model.Annotation
	if err := cursor.All(ctx, &annotations); err != nil {
		return nil, err
	}
	return annotations, nil
}

func (s *MongoStore) DeleteAnnotation(ctx context.Context, id string) error {
	_, err := s.annoCol.DeleteOne(ctx, bson.D{{Key: "_id", Value: id}})
	return err
}

func (s *MongoStore) CountAnnotationsByScene(ctx context.Context, sceneID string) (int64, error) {
	return s.annoCol.CountDocuments(ctx, bson.D{{Key: "scene_id", Value: sceneID}})
}

func (s *MongoStore) SaveCANLog(ctx context.Context, canLog *model.CANLog) error {
	_, err := s.canCol.InsertOne(ctx, canLog)
	return err
}

func (s *MongoStore) GetCANLog(ctx context.Context, sceneID string) (*model.CANLog, error) {
	var canLog model.CANLog
	err := s.canCol.FindOne(ctx, bson.D{{Key: "scene_id", Value: sceneID}}).Decode(&canLog)
	if err != nil {
		return nil, err
	}
	return &canLog, nil
}
