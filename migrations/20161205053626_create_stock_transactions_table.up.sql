CREATE TABLE IF NOT EXISTS Transactions (
	id int(11) NOT NULL AUTO_INCREMENT,
	userId int(11) NOT NULL,
	stockId int(11) NOT NULL,
	type enum('FromExchange', 'OrderFill', 'Mortgage', 'Dividend'),
	stockQuantity int(11) NOT NULL,
	price int(11) NOT NULL,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id),
	FOREIGN KEY (userId) REFERENCES Users(id),
	FOREIGN KEY (stockId) REFERENCES Stocks(id)
) AUTO_INCREMENT=1;