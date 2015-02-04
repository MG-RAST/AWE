#!/bin/sh -e
# AWE auto-start
#
# description: auto-starts AWE client
# processname: awe-client
# pidfile: /var/run/awe-client.pid
# logfile: /var/log/awe-client.log
# config: /etc/awe/awe-client.conf
 
NAME="awe-client"
PID_FILE="/var/run/${NAME}.pid"
LOG_FILE="/var/log/${NAME}.log"
CONF_FILE="/etc/awe/${NAME}.conf"

start() {
    AWE_OPTS="-conf $CONF_FILE"
    if [ -n "$1" ]; then
    	AWE_OPTS="-debug 3 $AWE_OPTS"
    	echo "Running in debug mode"
    fi
    echo -n "Starting $NAME... "
    if [ -f $PID_FILE ]; then
	    echo "is already running!"
    else
	    $NAME $AWE_OPTS > $LOG_FILE 2>&1 &
	    sleep 2
	    echo `ps -ef | grep -v grep | grep 'awe-client' | awk '{print $2}'` > $PID_FILE
	    echo "(Done)"
    fi
    return 0
}
 
stop() {
    echo -n "Stopping $NAME... "
    if [ -f $PID_FILE ]; then
	    PIDN=`cat $PID_FILE`
	    kill -TERM -$PIDN 2>&1
	    sleep 2
	    rm $PID_FILE
	    echo "(Done)"
    else
	    echo "can not stop, it is not running!"
    fi
    return 0
}

status() {
    if [ -f $PID_FILE ]; then
	    PIDN=`cat $PID_FILE`
	    echo "$NAME is running with pid $PIDN."
    else
	    echo "$NAME is not running."
    fi
    return 0
}

case "$1" in
    debug)
    	start 1
    	;;
    start)
	    start
	    ;;
    stop)
	    stop
	    ;;
    restart)
	    stop
	    sleep 5
	    start
	    ;;
    status)
	    status
	    ;;
    *)
	    echo "Usage: $0 (debug | start | stop | restart | status)"
	    exit 1
	    ;;
esac
