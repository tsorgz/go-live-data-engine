import time
import csv
import random
from datetime import datetime, timezone
import os
import string 

DATA_FILE = '/app/data/notes.csv'
INTERVAL = 0.1  

def ensure_file_exists():
    if not os.path.exists(DATA_FILE):
        with open(DATA_FILE, 'w', newline='') as f:
            writer = csv.writer(f)
            writer.writerow(['timestamp', 'user_id', 'note', ''])

def generate_data():
    while True:
        with open(DATA_FILE, 'a', newline='') as f:
            writer = csv.writer(f)
            timestamp = int(datetime.now(tz=timezone.utc).timestamp() * 1000)
            user_id = random.randint(1, 1000)
            note = ''.join([random.choice(string.ascii_letters) for _ in range(random.randint(4,64))])
            writer.writerow([timestamp, user_id, note, ''])
        
        time.sleep(random.random() * INTERVAL + 1e-5)

if __name__ == '__main__':
    ensure_file_exists()
    generate_data()