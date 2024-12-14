# **Auction System**

This project is an auction system developed in Go, using MongoDB as the database and Docker for environment management. It allows you to create, manage, and automatically close auctions based on a configurable interval.


## **Prerequisites**

Before starting, ensure you have the following installed on your machine:

- **Go**: To run the code locally.
- **Docker**: For container management.
- **Docker Compose**: To orchestrate the services.


## **Clone the Repository**

```bash
git clone <REPOSITORY-URL>
cd <PROJECT-NAME>
```


## **Environment Setup**

### 1. Configure environment variables
Create or edit the `.env` file in the project root with the following values:

- **`AUCTION_INTERVAL`**: Defines the default auction duration. Example: `120s`.
- **Other required variables**, such as database or authentication configurations, if applicable.

### 2. Start the containers
Use Docker Compose to build and start the services:

```bash
docker-compose up --build
```


## **Database Setup**

### 1. Connect to MongoDB
Once the containers are running, connect to MongoDB:

```bash
docker exec -it mongodb mongosh
```

### 2. Authenticate in MongoDB
```bash
use admin
db.auth("admin", "admin")
```

### 3. Create sample users in the database
Insert sample users into the database:

```bash
use auctions
db.users.insertMany([
  { _id: "ffe31076-e5a4-495a-8062-9a9506a13595", name: "Max" },
  { _id: "6eb46049-4005-4026-aedc-d8ce4d546034", name: "Bob" }
]);
exit
```


## **Running Tests**

Run the automated test to validate the automatic auction closure:

```bash
go test -timeout 30s -run ^TestAuctionAutoClosure$ fullcycle-auction_go/internal/infra/database/auction
```


## **Request Examples**

The project includes HTTP request examples in the file `api/request.http`. Use a tool like [Rest Client](https://marketplace.visualstudio.com/items?itemName=humao.rest-client) in Visual Studio Code to execute the examples.


## **Additional Documentation**

- The system behavior is configured via environment variables. Check the available values in the `.env` file.
- The core logic for automatically closing auctions is implemented in the `internal/infra/database/auction` package.

