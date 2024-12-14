package auction_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"fullcycle-auction_go/internal/entity/auction_entity"
	"fullcycle-auction_go/internal/infra/database/auction"
)

func TestAuctionAutoClosure(t *testing.T) {
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "mongo:5.0",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForLog("Waiting for connections").WithStartupTimeout(30 * time.Second),
	}

	mongoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("Failed to create MongoDB container: %v", err)
	}
	defer mongoContainer.Terminate(ctx)

	mappedPort, err := mongoContainer.MappedPort(ctx, "27017")
	if err != nil {
		log.Fatalf("Failed to get mapped port: %v", err)
	}

	hostIP, err := mongoContainer.Host(ctx)
	if err != nil {
		log.Fatalf("Failed to get host IP: %v", err)
	}

	uri := "mongodb://" + hostIP + ":" + mappedPort.Port()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("test_auctions")

	os.Setenv("AUCTION_INTERVAL", "5s")
	repo := auction.NewAuctionRepository(db, nil)

	auctionID := uuid.New().String()
	auctionEntity := &auction_entity.Auction{
		Id:          auctionID,
		ProductName: "Test Product",
		Description: "Test Description",
		Category:    "Test Category",
		Condition:   1,
		Status:      auction_entity.Active,
		Timestamp:   time.Now(),
	}
	_ = repo.CreateAuction(ctx, auctionEntity)

	time.Sleep(6 * time.Second)

	var result auction.AuctionEntityMongo
	err = repo.Collection.FindOne(ctx, bson.M{"_id": auctionID}).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, auction_entity.Completed, result.Status)
}
