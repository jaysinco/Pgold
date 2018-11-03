$(function () {
    Highcharts.setOptions({
        lang: {
            rangeSelectorZoom: ''
        },
        global: {
            useUTC: false
        },
        chart: {
            style: {
                fontFamily: 'Arial'
            }
        }
    });
    var nday = new Date();
    var year = nday.getFullYear();
    var month = nday.getMonth() + 1;
    var day = nday.getDate();
    var today = year + '-' + (month > 9 ? month : '0' + month) + "-" + (day > 9 ? day : '0' + day);
    document.getElementById('tick_date').value = today;
    drawPaperGoldTick();
    state = "kline";
});

var state = null;

function toggle() {
    if (state == "kline") {
        document.getElementById('toggler').style.backgroundColor = '#FF2D2D';
        drawPaperGoldKLine();
        state = "spline";
    } else if (state == "spline") {
        document.getElementById('toggler').style.backgroundColor = '#3E8CD0';
        drawPaperGoldTick();
        state = "kline";
    }
};

function drawPaperGoldTick() {
    var nday = new Date()
    var date = document.getElementById('tick_date').value.replace(/-/g, '/')
    var start = Math.floor(Date.parse(date + ' 00:00:00') / 1000);
    var end = Math.floor(Date.parse(date + ' 23:59:59') / 1000);
    $.getJSON('/papergold/price/tick/json/by/timestamp?start=' + start + '&end=' + end, function (data) {
        if (data.length == 0) {
            document.getElementById('pg_price').innerHTML = "NO DATA FOUND ON " + date
            document.getElementById('tick_date').style.display = '';
            drawRandPoem();
            return
        }
        var pgcs = [];
        var ymax = -1.0;
        var ymin = 9999.0;
        var scale = Math.ceil(data.length / (24 * 60 * 2) * 15)
        for (var i = 0; i < data.length; i += 1) {
            if (i % scale != 0 && (i != data.length - 1)) {
                continue
            }
            if (data[i].p > ymax) {
                ymax = data[i].p;
            };
            if (data[i].p < ymin) {
                ymin = data[i].p;
            };
            pgcs.push([
                data[i].t * 1000,
                data[i].p
            ]);
        }
        var ylimax = 0.0;
        var ylimin = 0.0;
        var W = 4;
        var R = 2.8 / 5.0;
        if (ymax - ymin < W) {
            ylimax = (W - (ymax - ymin)) * R + ymax;
            ylimin = ymin - (W - (ymax - ymin)) * (1 - R);
        } else {
            ylimax = 0.5 + ymax;
            ylimin = ymin - 0.5;
        }
        $('#pg_price').highcharts({
            chart: {
                zoomType: 'x',
                height: 400
            },
            credits: {
                enabled: false
            },
            title: {
                text: 'ICBC Paper Gold Price',
                style: {
                    fontSize: "20px",
                    //fontWeight: "bold",
                    fontFamily: "Arial",
                }
            },
            xAxis: {
                gridLineWidth: 1,
                type: 'datetime',
                dateTimeLabelFormats: {
                    millisecond: '%H:%M:%S.%L',
                    second: '%H:%M:%S',
                    minute: '%H:%M',
                    hour: '%H:%M',
                    day: '%m-%d',
                    week: '%m-%d',
                    month: '%Y-%m',
                    year: '%Y'
                },
                crosshair: true,
                minPadding: 0,
                maxPadding: 0,
                min: (start - 1000) * 1000,
                //tickPixelInterval: 50,
                minRange: (end - start + 1000) * 1000,
            },
            tooltip: {
                dateTimeLabelFormats: {
                    millisecond: '%H:%M:%S.%L',
                    second: '%H:%M:%S',
                    minute: '%H:%M',
                    hour: '%H:%M',
                    day: '%Y-%m-%d',
                    week: '%m-%d',
                    month: '%Y-%m',
                    year: '%Y'
                },
                style: {
                    fontSize: "11px",
                },
            },
            yAxis: {
                lineWidth: 2,
                opposite: true,
                labels: {
                    align: 'right',
                    x: -3,
                },
                crosshair: true,
                title: {
                    text: 'CNY',
                    style: {
                        fontFamily: "Arial",
                    }
                },
                max: ylimax,
                min: ylimin,
            },
            legend: {
                enabled: false
            },
            series: [{
                type: 'spline',
                name: '银行买入价',
                data: pgcs,
                threshold: null,
                lineWidth: 2,
                marker: {
                    radius: 1.5,
                },
                states: {
                    hover: {
                        lineWidth: 2,
                    }
                },
            }],
        });
        document.getElementById('tick_date').style.display = '';
        drawRandPoem();
    });
};

function drawPaperGoldKLine() {
    $.getJSON('/papergold/price/kline/json/all/day', function (data) {
        var pgklines = [];
        for (var i = 0; i < data.length; i += 1) {
            pgklines.push([
                data[i].t * 1000,
                data[i].o,
                data[i].h,
                data[i].l,
                data[i].c
            ]);
        }
        $('#pg_price').highcharts('StockChart', {
            credits: {
                enabled: false
            },
            chart: {
                height: 400
            },
            rangeSelector: {
                allButtonsEnabled: true,
                buttons: [{
                    type: 'month',
                    count: 1,
                    text: 'Day',
                    dataGrouping: {
                        forced: true,
                        units: [['day', [1]]]
                    }
                }, {
                    type: 'all',
                    text: 'Week',
                    dataGrouping: {
                        forced: true,
                        units: [['week', [1]]]
                    }
                }, {
                    type: 'all',
                    text: 'Month',
                    dataGrouping: {
                        forced: true,
                        units: [['month', [1]]]
                    }
                }],
                buttonTheme: {
                    width: 60
                },
                selected: 0
            },
            title: {
                text: 'ICBC Paper Gold Price',
                style: {
                    fontSize: "20px",
                    //fontWeight: "bold",
                    fontFamily: "Arial",
                }
            },
            tooltip: {
                pointFormatter: function () {
                    for (var i = 0; i < pgklines.length; i++) {
                        if (pgklines[i][0] == this.x) {
                            return "<span style=\"color:#FF6347\">●</span> 银行买入价<br/>" +
                                "最高: <b>" + pgklines[i][2].toFixed(2) + "</b><br/>开盘: <b>" + pgklines[i][1].toFixed(2) + "</b><br/>" +
                                "收盘: <b>" + pgklines[i][4].toFixed(2) + "</b><br/>最低: <b>" + pgklines[i][3].toFixed(2) + "</b><br/>" +
                                "涨幅: <b>" + ((pgklines[i][4] - pgklines[i][1]) / pgklines[i][1] * 100).toFixed(2) + "%</b>"

                        }
                    }
                },
                style: {
                    fontSize: "11px",
                },
            },
            xAxis: {
                gridLineWidth: 1,
                dateTimeLabelFormats: {
                    millisecond: '%H:%M:%S.%L',
                    second: '%H:%M:%S',
                    minute: '%H:%M',
                    hour: '%H:%M',
                    day: '%m-%d',
                    week: '%m-%d',
                    month: '%y-%m',
                    year: '%Y'
                },
            },
            yAxis: [{
                labels: {
                    align: 'right',
                    x: -3
                },
                title: {
                    text: 'CNY'
                },
                lineWidth: 2
            }],
            series: [{
                type: 'candlestick',
                name: 'Paper Gold',
                color: 'green',
                lineColor: 'green',
                upColor: 'red',
                upLineColor: 'red',
                navigatorOptions: {
                    color: 'Silver'
                },
                data: pgklines
            }]
        });
        document.getElementById('tick_date').style.display = 'none';
        drawRandPoem();
    });
};