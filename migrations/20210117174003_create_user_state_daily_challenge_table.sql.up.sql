CREATE TABLE IF NOT EXISTS UserState (
     `challengeId` int(11) UNSIGNED NOT NULL,
     `userId` int(11) UNSIGNED NOT NULL,
     `marketDay` int(11) UNSIGNED NOT NULL,
     `initialValue`bigint(11) UNSIGNED NOT NULL,
     `finalValue`bigint(11) UNSIGNED DEFAULT NULL,
     `isCompleted`BOOLEAN DEFAULT FALSE,
     FOREIGN KEY (challengeId) REFERENCES DailyChallenge(id),
     FOREIGN KEY (userId) REFERENCES Users(id)
);