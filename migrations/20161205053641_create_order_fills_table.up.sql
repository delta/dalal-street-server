CREATE TABLE IF NOT EXISTS OrderFills (
	transactionId int(11) UNSIGNED NOT NULL,
	bidId int(11) UNSIGNED NOT NULL,
	askId int(11) UNSIGNED NOT NULL,
	FOREIGN KEY (transactionId) REFERENCES Transactions(id),
	FOREIGN KEY (bidId) REFERENCES Bids(id),
	FOREIGN KEY (askId) REFERENCES Asks(id)
);
