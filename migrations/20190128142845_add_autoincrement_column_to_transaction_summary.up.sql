DROP TABLE IF EXISTS TransactionSummary;

CREATE TABLE IF NOT EXISTS TransactionSummary(
    id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	userId int(11) UNSIGNED NOT NULL,
    stockId int(11) UNSIGNED NOT NULL,
    stockQuantity bigint(11) SIGNED NOT NULL,
    price float(11,2) UNSIGNED NOT NULL,
    PRIMARY KEY (id),
    UNIQUE KEY NONCLUSTERED (userId, stockId),
	FOREIGN KEY (userId) REFERENCES Users(id),
    FOREIGN KEY (stockId) REFERENCES Stocks(id)
) AUTO_INCREMENT=1;