ALTER TABLE Transactions MODIFY type enum('FromExchangeTransaction', 'OrderFillTransaction', 'MortgageTransaction', 'DividendTransaction', 'OrderFeeTransaction', 'TaxTransaction', 'PlaceOrderTransaction', 'CancelOrderTransaction', 'ShortSellTransaction', 'IpoAllotmentTransaction');
