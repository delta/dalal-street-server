CREATE TABLE IF NOT EXISTS StockHistory (
	stockId int(11) UNSIGNED NOT NULL,
	stockPrice bigint(11) UNSIGNED NOT NULL,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	FOREIGN KEY (stockId) REFERENCES Stocks(id)
);
