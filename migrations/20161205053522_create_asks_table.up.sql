CREATE TABLE IF NOT EXISTS Asks (
	id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
	userId int(11) UNSIGNED NOT NULL,
	stockId int(11) UNSIGNED NOT NULL,
	orderType enum('Limit', 'Market', 'StopLoss', 'StopLossActive'),
	price bigint(11) UNSIGNED NOT NULL,
	stockQuantity bigint(11) UNSIGNED NOT NULL,
	stockQuantityFulFilled bigint(11) UNSIGNED NOT NULL,
	isClosed tinyint(1) NOT NULL,
	createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	updatedAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (id),
	FOREIGN KEY (userId) REFERENCES Users(id),
	FOREIGN KEY (stockId) REFERENCES Stocks(id)
) AUTO_INCREMENT=1;
