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
            document.getElementById('pg_price').innerHTML = "";
            alert("NO DATA FOUND ON " + date);
            document.getElementById('tick_date').style.display = '';
            drawRandPoem();
            return
        }
        var pgcs = [];
        var ymax = -1.0;
        var ymin = 9999.0;
        var scale = Math.ceil(data.length / (24 * 60 * 2) * 15);
        for (var i = 0; i < data.length; i += 1) {
            if (i % scale != 0 && (i != data.length - 1)) {
                continue;
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
        var ylimax = Math.ceil(ymax * 2) / 2 + 0.5;
        var ylimin = Math.floor(ymin * 2) / 2 - 0.5;
        var plotBandsList = [];
        var bandNum = Math.ceil(ylimax - ylimin) * 2;
        var bandSep = 0.5;
        for (var i = 0; i < bandNum; i++) {
            plotBandsList.push({
                from: ylimin + i * bandSep,
                to: ylimin + (i + 1) * bandSep,
                color: i % 2 == 0 ? 'rgba(68, 170, 213, 0.1)' : 'rgba(255, 255, 255,0.1)',
            })
        }
        $('#pg_price').highcharts({
            chart: {
                panning: false,
                pinchType: '',
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
                minRange: (end - start + 1000) * 1000,
            },
            tooltip: {
                backgroundColor: 'rgba(255, 255, 255, 0.8)',
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
                borderWidth: 1.5,
                borderRadius: 20,
                borderColor: 'rgba(50, 50, 50, 0.8)',
                shadow: true,
                split: false,
            },
            yAxis: {
                gridLineWidth: 0,
                lineWidth: 2,
                minTickInterval: 0.5,
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
                plotBands: plotBandsList,
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
                lineWidth: 3,
                //color: '#606060',
                marker: {
                    radius: 1.5,
                },
                states: {
                    hover: {
                        lineWidth: 3,
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
                panning: false,
                pinchType: '',
                height: 400,
            },
            rangeSelector: {
                allButtonsEnabled: true,
                buttons: [{
                    type: 'day',
                    count: 35,
                    text: 'Day',
                    dataGrouping: {
                        forced: true,
                        units: [['day', [1]]]
                    }
                }],
                buttonPosition: {
                    x: 119,
                    y: 8,
                },
                buttonTheme: {
                    width: 60
                },
                selected: 0,
                inputEnabled: false,
            },
            navigator: {
                margin: 20,
            },
            scrollbar: {
                minWidth: 22,
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
                backgroundColor: 'rgba(255, 255, 255, 0.5)',
                borderWidth: 0,
                borderRadius: 0,
                pointFormatter: function () {
                    for (var i = 0; i < pgklines.length; i++) {
                        if (pgklines[i][0] == this.x) {
                            return "<span style=\"color:#FF6347\">●</span> 银行买入价<br/>" +
                                "高: <b>" + pgklines[i][2].toFixed(2) + "</b> 开: <b>" + pgklines[i][1].toFixed(2) + "</b> " +
                                "幅: <b>" + ((pgklines[i][4] - pgklines[i][1]) / pgklines[i][1] * 100).toFixed(2) + "%</b><br/>" +
                                "低: <b>" + pgklines[i][3].toFixed(2) + "</b> 收: <b>" + pgklines[i][4].toFixed(2) + "</b>"
                        }
                    }
                },
                style: {
                    fontSize: "10px",
                },
                positioner: function () {
                    return { x: 0, y: 40 };
                },
                shadow: false,
                split: false,
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
                crosshair: true,
            },
            yAxis: [{
                labels: {
                    align: 'right',
                    x: -3
                },
                title: {
                    text: 'CNY'
                },
                lineWidth: 2,
                crosshair: true,
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