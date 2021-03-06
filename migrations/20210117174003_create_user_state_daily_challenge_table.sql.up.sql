CREATE TABLE IF NOT EXISTS UserState (
     id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
     challengeId int(11) UNSIGNED NOT NULL,
     userId int(11) UNSIGNED NOT NULL,
     initialValue bigint(11) SIGNED NOT NULL,
     finalValue bigint(11) SIGNED DEFAULT NULL,
     isCompleted BOOLEAN DEFAULT FALSE,
     isRewardClaimed BOOLEAN DEFAULT FALSE,
     PRIMARY KEY (id),
     FOREIGN KEY (challengeId) REFERENCES DailyChallenge(id),
     FOREIGN KEY (userId) REFERENCES Users(id)
) AUTO_INCREMENT=1;
