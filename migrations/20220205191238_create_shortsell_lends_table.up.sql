CREATE TABLE ShortSellLends(
    stockId int(11) UNSIGNED NOT NULL,
    userId int(11) UNSIGNED NOT NULL,
    stockQuantity int(11) UNSIGNED NOT NULL,
    isSquaredOff BOOLEAN DEFAULT FALSE,
    FOREIGN KEY (stockId) REFERENCES Stocks(id),
    FOREIGN KEY (userId) REFERENCES Users(id)
);
