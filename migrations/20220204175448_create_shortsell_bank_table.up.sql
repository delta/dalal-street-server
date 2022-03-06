CREATE TABLE ShortSellBank(
    stockId int(11) UNSIGNED NOT NULL UNIQUE,
    availableStocks int(11) UNSIGNED DEFAULT 0,
    PRIMARY KEY(stockId),
    FOREIGN KEY (stockId) REFERENCES Stocks(id)
);
