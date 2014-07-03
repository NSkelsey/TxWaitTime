function NetPane(selection) {
    var margin = {top: 30, right: 20, bottom: 20, left: 50},
    height = 240 - margin.top - margin.bottom,
    width = 800 - margin.right - margin.left,
    color = undefined,
    timeBack = -1; // defines a one minute window


    var chart = {
        data: {},
        draw: function draw(selection) {

            var svg = selection
                .attr("width", width + margin.right + margin.left)
                .attr("height", height + margin.top + margin.bottom);

            chart.g = svg.append("g")
                .attr("transform", "translate(" + margin.left + "," + margin.top + ")")
                .attr("width", width) 
                .attr("height", height);

            //
            var yDom = [34, 170, 350, 10000];
            var yRan = [height, 2*height/3, height/3, 0];

            // Build scales
            chart.yScale = d3.scale.linear()
                .domain(yDom)
                .range(yRan)
                .clamp(true);

            y = chart.yScale;

            sizeBox = svg.append("g")
                .attr("transform", "translate("+(margin.left/2)+","+margin.top+")")
                .attr("class", "scale-box")

            sizeBox.append("rect")
                .attr("class", "small")
                .attr("x", 0)
                .attr("y", y(yDom[1]) + 5)
                .attr("width", margin.left/2)
                .attr("height", y(yDom[1]) - y(yDom[2]) - 5);

            sizeBox.append("rect")
                .attr("class", "avg")
                .attr("x", 0)
                .attr("y", y(yDom[2]))
                .attr("width", margin.left/2)
                .attr("height", y(yDom[1]) - y(yDom[2]));
              
            sizeBox.append("rect")
                .attr("class", "large")
                .attr("x", 0)
                .attr("y", y(yDom[3]))
                .attr("width", margin.left/2)
                .attr("height", y(yDom[2]) - y(yDom[3]) - 5);

            var now = new Date();
            var past = d3.time.minute.offset(now, -1);
            var tScale = d3.time.scale()
                .domain(timeWindow())
                .range([0, width]);

            chart.tScale = tScale;

            var xAxis = d3.svg.axis()
                .scale(chart.tScale)
                .orient("bottom")
                .ticks(d3.time.seconds, 15);
            
            var yAxis = d3.svg.axis()
                .scale(chart.yScale)
                .orient("left")
                .ticks(2);

            svg.append("g")
                .attr("class", "y axis")
                .attr("transform", "translate("+margin.left+","+margin.top+")")
                .call(yAxis);
            

            var tOffset = "translate(" + margin.left + "," + (height + margin.top) + ")";
            var tAxisSel = svg.append("g")
                .attr("transform",  tOffset)
                .attr("class", "time axis")
                .call(xAxis)
            
            tick();

            function tick() {
                chart.tScale.domain(timeWindow());

                tAxisSel.transition()
                    .ease("linear")
                    .duration(500)
                    .call(xAxis)
                    .each("end", tick);
            }
            
        }
    }

    function timeWindow() {
        now = new Date();
        past = d3.time.minute.offset(now, timeBack)
        return [past, now]
    }

    function openLink(obj) {
        var url = "https://www.biteasy.com/testnet/transactions/" + obj.txid;
        window.open(url, '_blank');
    }

    chart.update = function(obj){
        var c = chart;
//        arrdata = Object.keys(c.data).map(function(k){ return c.data[k]; })
//        
//        var join = chart.g.selectAll(".elem")
//            .data(arrdata, function(d) { return id(d); })


        //join.attr("class", "update")

        // Item enter
        chart.g.append("circle").datum(obj)
            .attr("class", "tx")
            .on("click", openLink)
            .attr("r", 0)
            .attr("cx", function(d) { 
                d.start_x = (width - 20) + Math.random()*20;
                return d.start_x;
            })
            .attr("cy", function(d) { return c.yScale(d.size); })
            .style("fill", function(d) { return color(d.kind); })
            .attr("opacity", 0)
          .transition()
            .duration(function() { return 250 - 100 + 100*Math.random() })
            .style("opacity", 0.8)
            .attr("r", 7)
          .transition()
            .ease("linear")
            .duration(60 * 1000)
            .attr("cx", function(d) {
                var fin_x = width - d.start_x; 
                return fin_x;
            })
            .each("end", function(d) { 
                d3.select(this).call(exitMe, d);
            });

        function exitMe(select, datum) {
            select.transition()
                .duration(100)
                .style("opacity", 0)
              .remove();
            delete c.data[id(datum)];
        }

    }

    chart.addObject = function(obj){
        chart.data[id(obj)] = obj;
        chart.update(obj);
    }

    chart.color = function(_) {
        if (!arguments.length) return color;
        color = _; 
        return chart
    }

    function id(obj) {
        var type = obj["type"];
        if (type === "tx") {
            return obj.txid;
        } else if (type == "block") {
            return obj.hash;
        } else {
            console.log("bad type")
            return undefined
        }
    }


    return chart

}


