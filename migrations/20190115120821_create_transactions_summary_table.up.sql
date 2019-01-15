CREATE TABLE IF NOT EXISTS TransactionSummary(
	userId int(11) UNSIGNED NOT NULL,
    stockId int(11) UNSIGNED NOT NULL,
    stockQuantity bigint(11) SIGNED NOT NULL,
    price float(11,2) UNSIGNED NOT NULL,
	FOREIGN KEY (userId) REFERENCES Users(id),
    FOREIGN KEY (stockId) REFERENCES Stocks(id),
	PRIMARY KEY (userId, stockId)
	) AUTO_INCREMENT = 1;
