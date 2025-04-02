# Go Crypto Price

A Go-based cryptocurrency price tracking application that fetches and stores real-time data from the Kraken exchange. This application provides a RESTful API to access cryptocurrency trading data, historical information, and server status.

## Features

- Real-time cryptocurrency price tracking
- Automatic data collection every 5 minutes
- Historical data storage in SQLite database
- CSV export functionality
- RESTful API endpoints
- Docker support for easy deployment

## API Endpoints

### Server Status
- **GET** `/api/status`
  - Returns the current status of the Kraken exchange server
  - Useful for monitoring system health

### Trading Pairs
- **GET** `/api/pairs`
  - Returns the top 10 trading pairs by volume
  - Includes detailed information about each pair
  - Sorted by 24-hour trading volume

- **GET** `/api/pairs/:pair`
  - Returns detailed information about a specific trading pair
  - Replace `:pair` with the trading pair symbol (e.g., "BTCUSD")

### Historical Data
- **GET** `/api/historical`
  - Downloads historical data in CSV format
  - Optional query parameter: `date` (format: YYYY-MM-DD)
  - If no date is provided, returns the most recent data
  - CSV includes OHLCV (Open, High, Low, Close, Volume) data

### Database Data
- **GET** `/api/db`
  - Returns all stored data from the local SQLite database
  - Includes trading pairs, pair information, and historical data

## Installation

### Using Docker

1. Clone the repository:
```bash
git clone https://github.com/yourusername/Go-CryptoPrice.git
cd Go-CryptoPrice
```

2. Build and run using Docker Compose:
```bash
docker-compose up --build
```

### Manual Installation

1. Ensure you have Go installed on your system
2. Clone the repository
3. Install dependencies:
```bash
go mod download
```
4. Run the application:
```bash
go run main.go
```

## Data Storage

The application uses SQLite for data storage (`crypto.db`) and creates CSV files in the `csv/` directory for historical data exports.

## Project Structure

```
Go-CryptoPrice/
├── database/     # Database operations and models
├── handlers/     # HTTP request handlers
├── kraken/       # Kraken API client
├── models/       # Data models
├── main.go       # Application entry point
├── Dockerfile    # Docker configuration
└── docker-compose.yml
```

## License

This project is licensed under the AGPL-3 License - see the [LICENSE](LICENSE) file for details.
