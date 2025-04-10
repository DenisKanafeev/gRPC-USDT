CREATE TABLE IF NOT EXISTS rates (
    id SERIAL PRIMARY KEY,
    ask DECIMAL(10, 2) NOT NULL,
    bid DECIMAL(10, 2) NOT NULL,
    ask_amount DECIMAL(15, 2) NOT NULL,
    bid_amount DECIMAL(15, 2) NOT NULL,
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );
