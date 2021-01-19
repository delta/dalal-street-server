CREATE TABLE IF NOT EXISTS DailyChallenge(
    `id` int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
    `marketDay` int(11) UNSIGNED NOT NULL,
    `challengeType` enum('Cash','NetWorth','StockWorth','SpecificStock'),
    `value` bigint(11) UNSIGNED NOT NULL,
    `stockId` int(11) UNSIGNED DEFAULT NULL,
    PRIMARY KEY (id),
    FOREIGN KEY (stockId) REFERENCES Stocks(id),
    CONSTRAINT check_daily_challenge CHECK((challengeType='SpecificStock' AND stockId>0 ) OR (challengeType<>'SpecificStock' AND stockId=0))
) AUTO_INCREMENT=1;