import csv
import sys
import datetime

l = []
minint = -sys.maxint - 1
maxint = sys.maxint

# FIRST argument is stock id Second in file to be input
stock_id = sys.argv[1]
filename = sys.argv[2]
output_file = "Result_" + filename


# CSV Structure
# ['Date', 'Open', 'High', 'Low', 'Close', 'Adj Close', 'Volume']

# Required
# ['stockId', 'close', 'createdAt', 'intervalRecord', 'high', 'low', 'open', 'volume']

now = datetime.datetime.now()
def getTimeString(now, minutes):
    seconds = 0
    year = now.year
    month = now.month
    date = now.day
    hours = now.hour - 4
    hour_add = int(minutes / 60)
    minutes = minutes % 60
    hours += hour_add
    day_add = hours / 24
    hours = hours % 24
    date += day_add
    t = datetime.datetime(year, month, date, hours, minutes, seconds)
    return t.strftime('%Y-%m-%dT%H:%M:%SZ')

def checkpoint(n, line_count, l, final_list):
    if line_count % n == 0 and line_count != 0:
        print "Creating Row for", n
        open_value = l[line_count - n][6]
        low = maxint
        high = minint
        close_value = l[line_count - 1][1]
        volume = 0
        created_at = l[line_count - 1][2]
        for i in range (1,n):
            volume += float(l[line_count - i][7])
            if float(low) > float(l[line_count - i][5]):
                low = l[line_count - i][5]
            if float(high) < float(l[line_count - i][4]):
                high = l[line_count - i][4]
        temp_row = []
        temp_row.extend([stock_id, close_value, created_at, n, high, low, open_value, volume])
        final_list.append(temp_row)

def clean_data(row):
    for val in row:
        if val == "null":
            return False
    return True

with open(filename) as csv_file:
    csv_reader = csv.reader(csv_file, delimiter=',')
    line_count = 0
    for row in csv_reader:
        if line_count == 0:
            print row
            line_count += 1
        else:
            temp_row = []
            if clean_data(row):
                temp_row.extend([stock_id, row[4], getTimeString(now, line_count), 1, row[2], row[3], row[1], row[6]])
                l.append(temp_row)
            line_count += 1


line_count = 0
final_list = []

for row in l:
    temp_row = []
    temp_row.extend(row)
    final_list.append(temp_row)
    
    line_count += 1
    checkpoint(5,line_count,l, final_list)
    checkpoint(15,line_count,l, final_list)
    checkpoint(30,line_count,l, final_list)
    checkpoint(60,line_count,l, final_list)
            

with open(output_file, 'w') as csvFile:
    writer = csv.writer(csvFile)
    writer.writerows(final_list)

csvFile.close()