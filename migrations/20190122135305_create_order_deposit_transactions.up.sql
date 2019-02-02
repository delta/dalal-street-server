CREATE TABLE IF NOT EXISTS OrderDepositTransactions (
	orderId int(11) UNSIGNED NOT NULL,
    transactionId int(11) UNSIGNED NOT NULL,
    isAsk tinyint(1) NOT NULL,
    createdAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
	PRIMARY KEY (orderId, transactionId, isAsk),
    FOREIGN KEY (transactionId) REFERENCES Transactions(id)
);
