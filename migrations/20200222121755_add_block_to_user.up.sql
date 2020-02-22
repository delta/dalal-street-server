ALTER TABLE `Users` ADD `isBlocked` BOOL NOT NULL;
ALTER TABLE `Users` ADD `blockCount` int(11) NOT NULL DEFAULT 0;
