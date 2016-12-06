CREATE TABLE IF NOT EXISTS MarketEvents (
	id int(11) NOT NULL AUTO_INCREMENT,
	stockId int(11) NOT NULL,
	emotionScore int(11) NOT NULL,
	`text` text,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id),
	FOREIGN KEY (stockId) REFERENCES Stocks(id)
) AUTO_INCREMENT=1;