import matplotlib.pyplot as plt
import numpy as np
import sys
import time
import struct
import heapq
from sklearn import preprocessing

def load_data():
    dfile = open('pgmkt.dat', 'rb')
    num, = struct.unpack('<q', dfile.read(8))
    prices = []
    times = []
    for _ in range(num):
        timestamp, bankbuy = struct.unpack('<qf', dfile.read(12))
        prices.append(bankbuy)
        times.append(time.localtime(timestamp))
    dfile.close()
    return np.array(prices), times

pcs, tms = load_data()

def ftime(t):
    return time.strftime("%Y-%m-%d %H:%M:%S", t)

def match(start, window=60, keep_n=10, prices=pcs):
    end = start + window
    target = preprocessing.scale(prices[start:end])
    length = len(prices)
    record = []
    slim = start-window
    for i in range(slim):
        sys.stdout.write('\r{}/{}'.format(i, slim))
        bound = window+i
        if bound <= length:
            compared = preprocessing.scale(prices[i:bound])
            dist = np.linalg.norm(target-compared)
            dist_norm = 1.0/(1.0+dist)
            record.append([i, dist_norm])
    keep = heapq.nlargest(keep_n, record, key=lambda s: s[1])
    print('')
    for k in keep:
        print("[{:8d}]  {:.3f}".format(k[0], k[1]))

def check(r1, g1, window=60, mul=2, prices=pcs):
    r2 = r1 + window
    g2 = g1 + window
    plt.subplot(1, 2, 1)
    plt.plot(preprocessing.scale(prices[r1:r2]), 'r')
    plt.plot(preprocessing.scale(prices[g1:g2]), 'g')
    plt.subplot(2, 2, 2)
    plt.plot(prices[r1:r2+window*mul], 'r')
    for i in range(mul):
        plt.axvline((r2-r1)*(i+1))
    plt.subplot(2, 2, 4)
    plt.plot(prices[g1:g2+window*mul], 'g')
    for i in range(mul):
        plt.axvline((r2-r1)*(i+1))
    plt.show()
