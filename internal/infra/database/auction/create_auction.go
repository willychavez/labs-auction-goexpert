package auction

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/internal_error"
)

type AuctionEntityMongo struct {
	Id          string                          `bson:"_id"`
	ProductName string                          `bson:"product_name"`
	Category    string                          `bson:"category"`
	Description string                          `bson:"description"`
	Condition   auction_entity.ProductCondition `bson:"condition"`
	Status      auction_entity.AuctionStatus    `bson:"status"`
	EndDate     int64                           `bson:"end_date"`
	Timestamp   int64                           `bson:"timestamp"`
}

type AuctionStatusUpdater interface {
	UpdateAuctionStatus(auctionId string, status auction_entity.AuctionStatus)
}

type AuctionRepository struct {
	Collection           *mongo.Collection
	auctionDuration      time.Duration
	statusUpdateCallback AuctionStatusUpdater
}

func NewAuctionRepository(database *mongo.Database, statusUpdater AuctionStatusUpdater) *AuctionRepository {
	repo := &AuctionRepository{
		Collection:           database.Collection("auctions"),
		auctionDuration:      getAuctionDuration(),
		statusUpdateCallback: statusUpdater,
	}

	go repo.monitorAuctionClosures(context.Background())
	return repo
}

func (ar *AuctionRepository) CreateAuction(
	ctx context.Context,
	auctionEntity *auction_entity.Auction) *internal_error.InternalError {
	auctionEntity.EndDate = time.Now().Add(ar.auctionDuration)
	auctionEntityMongo := &AuctionEntityMongo{
		Id:          auctionEntity.Id,
		ProductName: auctionEntity.ProductName,
		Category:    auctionEntity.Category,
		Description: auctionEntity.Description,
		Condition:   auctionEntity.Condition,
		Status:      auctionEntity.Status,
		EndDate:     auctionEntity.EndDate.Unix(),
		Timestamp:   auctionEntity.Timestamp.Unix(),
	}
	_, err := ar.Collection.InsertOne(ctx, auctionEntityMongo)
	if err != nil {
		logger.Error("Error trying to insert auction", err)
		return internal_error.NewInternalServerError("Error trying to insert auction")
	}

	return nil
}

func (ar *AuctionRepository) monitorAuctionClosures(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Auction closure monitor stopped")
			return
		case <-ticker.C:
			ar.closeExpiredAuctions(ctx)
		}
	}
}

func (ar *AuctionRepository) closeExpiredAuctions(ctx context.Context) {
	filter := bson.M{
		"status":   auction_entity.Active,
		"end_date": bson.M{"$lte": time.Now().Unix()},
	}
	update := bson.M{
		"$set": bson.M{
			"status": auction_entity.Completed,
		},
	}

	cur, err := ar.Collection.Find(ctx, filter)
	if err != nil {
		logger.Error("Error finding expired auctions", err)
		return
	}
	defer cur.Close(ctx)

	var auctions []AuctionEntityMongo
	if err := cur.All(ctx, &auctions); err != nil {
		logger.Error("Error decoding auctions", err)
		return
	}

	_, err = ar.Collection.UpdateMany(ctx, filter, update)
	if err != nil {
		logger.Error("Error closing expired auctions", err)
		return
	}

	for _, auction := range auctions {
		if ar.statusUpdateCallback != nil {
			ar.statusUpdateCallback.UpdateAuctionStatus(auction.Id, auction_entity.Completed)
		}
	}

	logger.Info(fmt.Sprintf("Closed %d expired auctions", len(auctions)))
}

func (ar *AuctionRepository) SetStatusUpdateCallback(updater AuctionStatusUpdater) {
	ar.statusUpdateCallback = updater
}

func getAuctionDuration() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Minute * 5
	}

	return duration
}
