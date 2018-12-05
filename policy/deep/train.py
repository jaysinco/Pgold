import matplotlib.pyplot as plt  
import numpy as np
import struct
import datetime
import mxnet as mx
import logging

logging.getLogger().setLevel(logging.DEBUG)

def view(sample_list):
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


f = open('raw_data.dp', 'rb')
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

range_count = 10
train_test_ratio = 5/1
total = len(samples) - range_count + 1
total_x = np.zeros(shape=(total, range_count))
total_y = np.zeros(shape=(total))

for i in range(total):
    total_y[i] = samples[i+range_count-1][2].index(1)
    for j in range(range_count):
        total_x[i][j] = samples[i+j][1]

train_n = int(total/(train_test_ratio+1)*train_test_ratio)
train_x = total_x[:train_n]
train_y = total_y[:train_n]
test_x = total_x[train_n:]
test_y = total_y[train_n:]

batch_size = 100
ctx = mx.cpu()
train_iter = mx.io.NDArrayIter(train_x, train_y, batch_size, shuffle=True)
test_iter = mx.io.NDArrayIter(test_x, test_y, batch_size)

data = mx.sym.var('data')
fc1  = mx.sym.FullyConnected(data=data, num_hidden=128)
act1 = mx.sym.Activation(data=fc1, act_type="relu")
fc2  = mx.sym.FullyConnected(data=act1, num_hidden=64)
act2 = mx.sym.Activation(data=fc2, act_type="relu")
fc3  = mx.sym.FullyConnected(data=act2, num_hidden=3)
mlp  = mx.sym.SoftmaxOutput(data=fc3, name='softmax')

mlp_model = mx.mod.Module(symbol=mlp, context=ctx)
# mlp_model.bind(data_shapes=train_iter.provide_data, label_shapes=train_iter.provide_label)
# mlp_model.init_params(initializer=mx.init.Xavier(magnitude=2.))

def train():
    mlp_model.fit(train_iter,  # train data
                eval_data=test_iter,  # validation data
                optimizer='sgd',  # use SGD to train
                optimizer_params={'learning_rate':0.01},  # use fixed learning rate
                eval_metric='acc',  # report accuracy during training
                batch_end_callback = mx.callback.Speedometer(batch_size, 1000), # output progress for each 100 data batches
                num_epoch=1)

# mlp_model.score(mx.io.NDArrayIter(train_x[:300], train_y[:300], batch_size), mx.metric.Accuracy())
