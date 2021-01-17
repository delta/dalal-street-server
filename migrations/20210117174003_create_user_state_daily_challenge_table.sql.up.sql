CREATE TABLE IF NOT EXISTS UserState (
     `marketDate` DATE NOT NULL,
     `userId` int(11) UNSIGNED NOT NULL,
     `initialValue`bigint(11) UNSIGNED NOT NULL,
     `currentValue`bigint(11) UNSIGNED NOT NULL,
     FOREIGN KEY (marketDate) REFERENCES DailyChallenge(marketDate),
     FOREIGN KEY (userId) REFERENCES Users(id)
);