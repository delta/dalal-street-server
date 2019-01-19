ALTER TABLE MortgageDetails 
ADD COLUMN mortgagePrice int(11) NOT NULL,
DROP PRIMARY KEY,
DROP COLUMN id,
ADD PRIMARY KEY(userId, stockId, mortgagePrice);
