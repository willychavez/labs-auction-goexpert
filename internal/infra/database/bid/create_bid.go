package bid

import (
	"context"
	"os"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"fullcycle-auction_go/configuration/logger"
	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/entity/bid_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
	"fullcycle-auction_go/internal/internal_error"
)

type BidEntityMongo struct {
	Id        string  `bson:"_id"`
	UserId    string  `bson:"user_id"`
	AuctionId string  `bson:"auction_id"`
	Amount    float64 `bson:"amount"`
	Timestamp int64   `bson:"timestamp"`
}

type BidRepository struct {
	Collection        *mongo.Collection
	AuctionRepository *auction.AuctionRepository
	auctionInterval   time.Duration
	auctionStatusMap  sync.Map // Replaces map[string]auction_entity.AuctionStatus
	auctionEndTimeMap sync.Map // Replaces map[string]time.Time
}

func NewBidRepository(database *mongo.Database, auctionRepository *auction.AuctionRepository) *BidRepository {
	return &BidRepository{
		auctionInterval:   getAuctionInterval(),
		Collection:        database.Collection("bids"),
		AuctionRepository: auctionRepository,
	}
}

func (bd *BidRepository) CreateBid(
	ctx context.Context,
	bidEntities []bid_entity.Bid) *internal_error.InternalError {
	var wg sync.WaitGroup
	for _, bid := range bidEntities {
		wg.Add(1)
		go func(bidValue bid_entity.Bid) {
			defer wg.Done()

			auctionStatus, okStatus := bd.getAuctionStatus(bidValue.AuctionId)
			auctionEndTime, okEndTime := bd.getAuctionEndTime(bidValue.AuctionId)

			bidEntityMongo := &BidEntityMongo{
				Id:        bidValue.Id,
				UserId:    bidValue.UserId,
				AuctionId: bidValue.AuctionId,
				Amount:    bidValue.Amount,
				Timestamp: bidValue.Timestamp.Unix(),
			}

			if okEndTime && okStatus {
				now := time.Now()
				if auctionStatus == auction_entity.Completed || now.After(auctionEndTime) {
					return
				}

				if _, err := bd.Collection.InsertOne(ctx, bidEntityMongo); err != nil {
					logger.Error("Error trying to insert bid", err)
					return
				}

				return
			}

			auctionEntity, err := bd.AuctionRepository.FindAuctionById(ctx, bidValue.AuctionId)
			if err != nil {
				logger.Error("Error trying to find auction by id", err)
				return
			}
			if auctionEntity.Status == auction_entity.Completed {
				return
			}

			bd.setAuctionStatus(bidValue.AuctionId, auctionEntity.Status)
			bd.setAuctionEndTime(bidValue.AuctionId, auctionEntity.Timestamp.Add(bd.auctionInterval))

			if _, err := bd.Collection.InsertOne(ctx, bidEntityMongo); err != nil {
				logger.Error("Error trying to insert bid", err)
				return
			}
		}(bid)
	}
	wg.Wait()
	return nil
}

func (bd *BidRepository) getAuctionStatus(auctionId string) (auction_entity.AuctionStatus, bool) {
	value, ok := bd.auctionStatusMap.Load(auctionId)
	if !ok {
		return auction_entity.AuctionStatus(0), false
	}
	return value.(auction_entity.AuctionStatus), true
}

func (bd *BidRepository) setAuctionStatus(auctionId string, status auction_entity.AuctionStatus) {
	bd.auctionStatusMap.Store(auctionId, status)
}

func (bd *BidRepository) getAuctionEndTime(auctionId string) (time.Time, bool) {
	value, ok := bd.auctionEndTimeMap.Load(auctionId)
	if !ok {
		return time.Time{}, false
	}
	return value.(time.Time), true
}

func (bd *BidRepository) setAuctionEndTime(auctionId string, endTime time.Time) {
	bd.auctionEndTimeMap.Store(auctionId, endTime)
}

func (bd *BidRepository) UpdateAuctionStatus(auctionId string, status auction_entity.AuctionStatus) {
	bd.setAuctionStatus(auctionId, status)
}

func getAuctionInterval() time.Duration {
	auctionInterval := os.Getenv("AUCTION_INTERVAL")
	duration, err := time.ParseDuration(auctionInterval)
	if err != nil {
		return time.Minute * 5
	}

	return duration
}
