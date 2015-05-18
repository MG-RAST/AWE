#!/bin/sh -e
# AWE auto-start
#
# description: auto-starts AWE server
# processname: awe-server
# pidfile: /var/run/awe-server.pid
# logfile: /var/log/awe-server.log
# config: /etc/awe/awe-server.conf
 
NAME="awe-server"
LOG_FILE="/var/log/${NAME}.log"
PID_FILE="/etc/awe/data/pidfile"
CONF_FILE="/etc/awe/${NAME}.conf"

start() {
    AWE_OPTS="-recover -conf $CONF_FILE"
    if [ -n "$1" ]; then
    	AWE_OPTS="-debuglevel 3 $AWE_OPTS"
    	echo "Running in debug mode"
    fi
    echo -n "Starting $NAME... "
    if [ -f $PID_FILE ]; then
	    echo "is already running!"
    else
	    $NAME $AWE_OPTS > $LOG_FILE 2>&1 &
	    sleep 1
	    echo "(Done)"
    fi
    return 0
}
 
stop() {
    echo -n "Stopping $NAME... "
    if [ -f $PID_FILE ]; then
	    PIDN=`cat $PID_FILE`
	    kill $PIDN 2>&1
	    sleep 1
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
	    PSTAT=`ps -p $PIDN | grep -v -w 'PID'`
	    if [ -z "$PSTAT" ]; then
	        echo "$NAME has pidfile ($PIDN) but is not running."
	    else
	        echo "$NAME is running with pid $PIDN."
	    fi
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
