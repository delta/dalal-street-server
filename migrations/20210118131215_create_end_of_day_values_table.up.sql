CREATE TABLE IF NOT EXISTS EndOfDayValues (
    id int(11) UNSIGNED NOT NULL AUTO_INCREMENT,
    userId int(11) UNSIGNED NOT NULL,
    cash bigint(11) UNSIGNED NOT NULL,
    debt bigint(11) UNSIGNED NOT NULL,
    stockWorth bigint(11) SIGNED NOT NULL,
    totalWorth bigint(11) SIGNED NOT NULL,
    FOREIGN KEY (userId) REFERENCES Users(id),
    PRIMARY KEY (id)
) AUTO_INCREMENT=1;
