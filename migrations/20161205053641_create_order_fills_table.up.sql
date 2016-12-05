CREATE TABLE IF NOT EXISTS OrderFils (
	transactionId int(11) NOT NULL,
	bidId int(11) NOT NULL,
	askId int(11) NOT NULL,
	FOREIGN KEY (transactionId) REFERENCES StockTransactions(id),
	FOREIGN KEY (bidId) REFERENCES Bids(id),
	FOREIGN KEY (askId) REFERENCES Asks(id)
);