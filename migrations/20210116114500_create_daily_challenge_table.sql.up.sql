CREATE TABLE IF NOT EXISTS DailyChallenge(
    `marketDate` DATE NOT NULL UNIQUE,
    `challengeType` enum('Cash','NetWorth','StockWorth','SpecificStock'),
    `value` bigint(11) UNSIGNED NOT NULL,
    `stockId` int(11) UNSIGNED DEFAULT NULL,
    PRIMARY KEY (marketDate),
    FOREIGN KEY (stockId) REFERENCES Stocks(id),
    CONSTRAINT check_daily_challenge CHECK((challengeType='SpecificStock' AND stockId>0 ) OR (challengeType<>'SpecificStock' AND stockId=0))
);