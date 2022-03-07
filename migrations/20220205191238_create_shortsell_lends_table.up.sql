CREATE TABLE ShortSellLends(
    id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
    stockId int(11) UNSIGNED NOT NULL,
    userId int(11) UNSIGNED NOT NULL,
    stockQuantity int(11) UNSIGNED NOT NULL,
    isSquaredOff BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (id),
    FOREIGN KEY (stockId) REFERENCES Stocks(id),
    FOREIGN KEY (userId) REFERENCES Users(id)
)AUTO_INCREMENT=1;
