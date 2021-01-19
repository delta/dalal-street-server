CREATE TABLE IF NOT EXISTS DailyLeaderboard (
    id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
    userId int(11) UNSIGNED NOT NULL,
    cash bigint(11) UNSIGNED NOT NULL,
    `rank` int(11) UNSIGNED NOT NULL,
    debt bigint(11) UNSIGNED NOT NULL,
    stockWorth bigint(11) SIGNED NOT NULL,
    totalWorth bigint(11) SIGNED NOT NULL,
    updatedAt varchar(255) NOT NULL DEFAULT "0000-00-00T00:00:00+05:30",
    userName varchar(255) NOT NULL,
    `isBlocked` BOOL NOT NULL DEFAULT false,
    FOREIGN KEY (userId) REFERENCES Users(id),
    PRIMARY KEY (id)
) AUTO_INCREMENT=1;
