import matplotlib.pyplot as plt  
import struct
import datetime

f = open('train.dat', 'rb')
raw = f.read()
f.close()
size, = struct.unpack("<i", raw[:4])

samples = []

action_buy = 1 << 0
action_sell = 1 << 1
action_hold = 1 << 2

for i in range(size):
    timestamp, pricediff, action = struct.unpack("<qfB", raw[4+13*i:4+13*(i+1)])
    realtime = datetime.datetime.fromtimestamp(timestamp)
    if action == action_buy:
        action_str = "buy"
        action_matrix = [1, 0, 0]
    elif action == action_sell:
        action_str = "sell"
        action_matrix = [0, 1, 0]
    elif action == action_hold:
        action_str = "hold"
        action_matrix = [0, 0, 1]
    # print("{} | {:5.2f} | {}".format(realtime.strftime('%Y-%m-%d %H:%M:%S'), pricediff, action_str))
    samples.append([realtime, pricediff, action_matrix])

print("{} sample to read".format(size))


def view_sample(sample_list):
    old_action = ""
    xbuffer = []
    ybuffer = []
    price = 0
    for index, sample in enumerate(sample_list):
        price += sample[1]
        current_action = "buy" if sample[2][0] == 1 else ("sell" if sample[2][1] == 1 else "hold")
        if current_action != old_action:
            if old_action != "":
                xbuffer.append(index)
                ybuffer.append(price)
                plt.plot(xbuffer, ybuffer, "red" if old_action == "buy" else ("green" if old_action == "sell" else "gray"), 
                    linewidth=2 if old_action != "hold" else 1)
                xbuffer = []
                ybuffer = []
            old_action = current_action
        xbuffer.append(index)
        ybuffer.append(price)
    if len(xbuffer) != 0:
        plt.plot(xbuffer, ybuffer, "red" if old_action == "buy" else ("green" if old_action == "sell" else "gray"), 
                    linewidth=2 if old_action != "hold" else 1)
    plt.show()