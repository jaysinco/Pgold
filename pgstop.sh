kill -9 $(ps -e|grep pgbrain|awk '{print $1}')
kill -9 $(ps -e|grep pgsvr  |awk '{print $1}')
kill -9 $(ps -e|grep pgmkt  |awk '{print $1}')

