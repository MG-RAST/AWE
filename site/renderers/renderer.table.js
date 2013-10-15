/*
  Table Renderer

  Displays a browsable, filterable table with clickable cells / rows.

  Options

  target (HTML Container Element)
      Element to render the table in.

  width (INT)
      Width of the table.

  height (INT)
      Height of the table.

  rows_per_page (INT)
      The maximum number of table rows to be displayed at a time. Default is 10.

  sortcol (INT)
      Zero based index of the row the table should be sorted by. Default is 0.

  sorted (BOOLEAN)
      Enables / disabled initial sorting of the table by the sortcol. Default is false.
  
  offset (INT)
      Initial first row to display. Default is 0.

  invisible_columns (HASH)
      Hash of column indices pointing at 1. Columns in this hash are not displayed.

  disable_sort (HASH)
      Hash of column indices pointing at 1. Columns in this hash can not be sorted.

  sorttype (HASH)
      Hash of column indices pointing at a sorttype. A sorttype can be either string or number.

  filter_autodetect (BOOLEAN)
      If set to false will try to detect which filter type is most appropriate for each column. Default is false.

  filter_autodetect_select_max (INT)
      Maximum number of distinct entries in a column that will still autodetec the column filter as a select box. Default is 10.

  editable (HASH of BOOLEAN)
      The key of the hash is the column index. If set to true, clicking a cell in this column will display an input field allowing to change the content of the cell.

  sort_autodetect (BOOLEAN)
      If set to true will try to detect which sorttype is appropriate for each column. Default is false.

  filter (HASH)
      Hash of column indices pointing at filter objects. A filter object has the properties
        searchword - the current entry in the search field
        case_sensitive - boolean to turn on / off case sensitivity in filtering
        operator - list of operators available in this filter
        active_operator - selected operator
        type - text or select

  hide_options (BOOLEAN)
      Turns display of the options button on and off. Default is false (the option button is visible).

  onclick (FUNCTION)
      The function to be called when the table is clicked. This function will be passed the parameters (as an ordered list)
        clicked_row - array of contents of the cells of the clicked row
        clicked_cell - content of the clicked cell
        clicked_row_index - zero based index of the clicked row
        clicked_cell_index - zero based index of the clicked cell

  edit_callback (FUNCTION)
      The function to be called when a cell is edited. This function is passed the table data. It is organised as a list of hashes with each column name pointing at the value of the cell.

  synchronous (BOOLEAN)
      This is true by default. If set to false, the table expects its data to be set, filtered and browsed externally. It will issue a callback to the navigation callback function on any of those events, expecting an external data update.

  navigation_callback (FUNCTION)
      The function to be called when a navigation / filter action is issued and the table is in asynchronous state (synchronous set to false). It will be passed either a string ("previous", "next", "first", "last") or an object that can contain one of the following structures:
        sort: { sort: $fieldname, dir: [ "asc" | "desc" ] }
        query: [ { searchword: $filter_value, field: $column_name_to_search, comparison: $comparison_operator }, ... ]
        goto: $row_index
        limit: $number_of_rows_per_page
    
*/
(function () {
    var renderer = Retina.Renderer.extend({
      about: {
	name: "table",
	    title: "Table",
            author: "Tobias Paczian",
            version: "1.0",
            requires: [],
            defaults: {
		'width': null,
		'height': null,
		'rows_per_page': 10,
		'sortcol': 0,
		'sorted': false,
		'offset': 0,
		'invisible_columns' : {},
		'disable_sort': {},
		'sortdir': 'asc',
		'sorttype': {},
		'filter_autodetect': false,
		'filter_autodetect_select_max': 10,
		'sort_autodetect': false,
		'filter': {},
		'hide_options': false,
		'filter_changed': false,
		'editable': {},
		'edit_callback': null,
		'navigation_callback': null,
		'target': 'table_space',
		'data': 'exampleData()',
		'synchronous': true,
	    }
      },
	exampleData: function () {
	    return {
		data: [ ["a1", "b1", "c1"],
			["a3", "b2", "c2"],
			["a4", "b3", "c3"],
			["a2", "b4", "c4"],
			["a1", "b1", "c1"],
			["a3", "b2", "c2"],
			["a4", "b3", "c3"],
			["a2", "b4", "c4"],
			["a1", "b1", "c1"],
			["a3", "b2", "c2"],
			["a4", "b3", "c3"],
			["a2", "b4", "c4"],
			["a1", "b3", "c1"],
			["a3", "b2", "c2"],
			["a4", "b3", "c3"],
			["a2", "b4", "c4"],
			["a5", "b5", "c5"] ],
		header: ["column A", "column B", "column C"]
	    };
        },
	update_visible_columns: function (index) {
	    renderer = Retina.RendererInstances.table[index];
	    var t = document.getElementById('table_colsel_table_'+index);
	    var r = t.firstChild.childNodes;
	    var inv = {};
	    for (i=0;i<r.length;i++) {
		if (! r[i].firstChild.firstChild.checked) {
		    inv[i] = 1;
		}
	    }
	    renderer.settings.invisible_columns = inv;
	    renderer.render();
	},
	render: function () {
	    renderer = this;
	    var index = renderer.index;
	    
	    renderer.settings.target.innerHTML = "";
	    
	    // check if we have a header, otherwise interpret the first line as the header
	    if (renderer.settings.data.length) {
		renderer.settings.data = { header: renderer.settings.data[0], data: renderer.settings.data };
		renderer.settings.data.data.shift();
	    }
	    
	    // if a header has already been initialized, don't touch it again
	    var header;
	    if (renderer.settings.header) {
		header = renderer.settings.header;
	    } else {
		header = renderer.settings.data.header;
		if (!renderer.settings.data.header) {
		    header = renderer.settings.data.data.shift();
		}
		renderer.settings.header = header;
		renderer.settings.data.header = null;
	    }
	    
	    // check if we have already parsed the data
	    var tdata = [];
	    if (renderer.settings.tdata) {
		tdata = renderer.settings.tdata;
	    } else {
		
		// the data has not been parsed, do it now
		for (i=0;i<renderer.settings.data.data.length; i++) {
		    tdata[tdata.length] = {};
		    for (h=0;h<renderer.settings.data.data[i].length;h++) {
			tdata[tdata.length - 1][header[h]] = renderer.settings.data.data[i][h] || "";
		    }
		}
		renderer.settings.tdata = tdata;
		renderer.settings.data.data = null;
	    }
	    
	    // if we are to auto determine sort functions, do so
	    if (renderer.settings.sort_autodetect) {
		for (var i=0; i<header.length; i++) {
		    if (!renderer.settings.sorttype[i]) {
			if (typeof(tdata[0][header[i]].replace) != 'function') {
			    renderer.settings.sorttype[i] = "number";
			} else {
			    var testval = tdata[0][header[i]].replace(/<(.|\n)*?>/g, "");
			    if (isNaN(parseFloat(testval))) {
				renderer.settings.sorttype[i] = "string";
			    } else {
				renderer.settings.sorttype[i] = "number";
			    }
			}
		    }
		}
	    }
	    
	    // create filter elements
	    var filter = renderer.settings.filter;
	    var filter_present = false;
	    for (i in filter) {
		if (filter.hasOwnProperty(i)) {
		    if (filter[i].hasOwnProperty('searchword') && filter[i].searchword.length > 0) {
			filter_present = true;
			break;
		    }
		}
	    }

	    // check for data filtering
	    if (filter_present && renderer.settings.synchronous) {
		var newdata = [];
		if (renderer.settings.filter_changed) {
		    renderer.settings.offset = 0;
		    for (i in filter) {
			var re;
			if (filter[i].case_sensitive) {
			    re = new RegExp(filter[i].searchword);
			} else {
			    re = new RegExp(filter[i].searchword, "i");
			}
			filter[i].re = re;
			if (typeof(filter[i].searchword) != "undefined" && filter[i].searchword.length > 0 &&filter[i].operator && filter[i].operator[filter[i].active_operator] == "><") {
			    filter[i].minmax = filter[i].searchword.split(",");
			    if (filter[i].minmax.length != 2) {
				alert("'"+filter[i].searchword + "' is not a valid inclusive range.\nRanges must be noted as the minimum\nand the maximum range, separated by ','\ni.e. '-2.1, 5.2'");
				filter[i].searchword = "";
			    }
			}
		    }
		    for (h=0; h<tdata.length; h++) {
			var pass = 1;
			for (i in filter) {
			    if (typeof(filter[i].searchword) != "undefined" && filter[i].searchword.length > 0) {
				if (filter[i].operator) {
				    switch (filter[i].operator[filter[i].active_operator]) {
				    case "=":
					if (tdata[h][header[i]] != filter[i].searchword) {
					    pass = 0;
					}
					break;
				    case ">":
					if (parseFloat(tdata[h][header[i]]) <= parseFloat(filter[i].searchword)) {
					    pass = 0;
					}
					break;
				    case "<":
					if (parseFloat(tdata[h][header[i]]) >= parseFloat(filter[i].searchword)) {
					    pass = 0;
					}
					break;				      
				    case "><":
					if (parseFloat(tdata[h][header[i]]) > parseFloat(filter[i].minmax[1]) || parseFloat(tdata[h][header[i]]) < parseFloat(filter[i].minmax[0])) {
					    pass = 0;
					}
					break;
				    }
				} else {
				    if (! tdata[h][header[i]].match(filter[i].re)) {
					pass = 0;
				    }
				}
				if (pass == 0) {
				    break;
				}
			    }
			}
			if (pass) {
			    newdata.push(tdata[h]);
			}
		    }
		} else {
		    newdata = renderer.settings.filtered_data;
		}
		renderer.settings.filter_changed = false;
		renderer.settings.filtered_data = newdata;
		tdata = newdata;
	    }
	    
	    // initialize the options
	    var offset = renderer.settings.offset;
	    var rows = (renderer.settings.rows_per_page < 0) ? tdata.length : renderer.settings.rows_per_page;
	    var sortcol = renderer.settings.sortcol;
	    var sortdir = renderer.settings.sortdir;
	    var sorttype = renderer.settings.sorttype;
	    var target = renderer.settings.target;
	    	    
	    // check width and height
	    var defined_width = "";
	    if (renderer.settings.width) {
		defined_width = "width: " + renderer.settings.width + "px; ";
	    }
	    var defined_height = "";
	    if (renderer.settings.height) {
		defined_height = "height: " + renderer.settings.height + "px; ";
	    }
	    
	    // create the actual table header
	    var table_element = document.createElement("table");
	    table_element.setAttribute("class", "table table-striped table-bordered table-condensed");
	    table_element.setAttribute("style", "margin-bottom: 2px;");
	    var thead = document.createElement("thead");
	    var tr = document.createElement("tr");
	    for (i=0;i<header.length;i++) {
		
		// check if this column is visible
		if (! renderer.settings.invisible_columns[i]) {
		    
		    // create sorting elements
		    var asc = document.createElement("i");
		    asc.setAttribute("class", "icon-chevron-down");
		    asc.setAttribute("title", "sort ascending");
		    var desc = document.createElement("i");
		    desc.setAttribute("class", "icon-chevron-up");
		    desc.setAttribute("title", "sort descending");
		    if (i == sortcol) {
			if (sortdir=='asc') {
			    asc.setAttribute("class", "icon-chevron-down icon-white");
			    asc.setAttribute("title", "current sorting: ascending");
			    desc.setAttribute("style", "cursor: pointer;");
			    desc.i = i;
			    desc.onclick = function () {
				Retina.RendererInstances.table[index].settings.sortcol = this.i;
				Retina.RendererInstances.table[index].settings.sortdir = 'desc';
				if (typeof renderer.settings.navigation_callback == "function") {
				    renderer.settings.navigation_callback({'sort': Retina.RendererInstances.table[index].settings.header[this.i] , 'dir': 'desc'});
				} else {
				    Retina.RendererInstances.table[index].settings.sorted = false;
				    Retina.RendererInstances.table[index].render();
				}
			    }
			} else {
			    desc.setAttribute("class", "icon-chevron-up icon-white");
			    desc.setAttribute("title", "current sorting: descending");
			    asc.setAttribute("style", "cursor: pointer;");
			    asc.i = i;
			    asc.onclick = function () {
				Retina.RendererInstances.table[index].settings.sortcol = this.i;
				Retina.RendererInstances.table[index].settings.sortdir = 'asc';
				if (typeof renderer.settings.navigation_callback == "function") {
				    renderer.settings.navigation_callback({'sort': Retina.RendererInstances.table[index].settings.header[this.i] , 'dir': 'asc'});
				} else {
				    Retina.RendererInstances.table[index].settings.sorted = false;
				    Retina.RendererInstances.table[index].render();
				}
			    }
			}
		    } else {
			asc.setAttribute("style", "cursor: pointer;");
			asc.i = i;
			asc.onclick = function () {
			    Retina.RendererInstances.table[index].settings.sortcol = this.i;
			    Retina.RendererInstances.table[index].settings.sortdir = 'asc';
			    if (typeof renderer.settings.navigation_callback == "function") {
				renderer.settings.navigation_callback({'sort': Retina.RendererInstances.table[index].settings.header[this.i] , 'dir': 'asc'});
			    } else {
				Retina.RendererInstances.table[index].settings.sorted = false;
				Retina.RendererInstances.table[index].render();
			    }
			}
			desc.setAttribute("style", "cursor: pointer;");
			desc.i = i;
			desc.onclick = function () {
			    Retina.RendererInstances.table[index].settings.sortcol = this.i;
			    Retina.RendererInstances.table[index].settings.sortdir = 'desc';
			    if (typeof renderer.settings.navigation_callback == "function") {
				renderer.settings.navigation_callback({'sort': Retina.RendererInstances.table[index].settings.header[this.i] , 'dir': 'desc'});
			    } else {
				Retina.RendererInstances.table[index].settings.sorted = false;
				Retina.RendererInstances.table[index].render();
			    }
			}
		    }
		    
		    // check for filter autodetection
		    if (renderer.settings.filter_autodetect) {
			if (! renderer.settings.filter[i]) {
			    renderer.settings.filter[i] = { type: "text" };
			    if (renderer.settings.sorttype[i] == "number") {
				renderer.settings.filter[i].operator = [ "=", "<", ">", "><" ];
				renderer.settings.filter[i].active_operator = 0;
			    }
			    var selopts = [];
			    var numopts = 0;
			    for (h=0;h<tdata.length;h++) {				  
				if (! selopts[tdata[h][header[i]]]) {
				    numopts++;
				}
				selopts[tdata[h][header[i]]] = 1;
			    }
			    if (numopts <= renderer.settings.filter_autodetect_select_max) {
				renderer.settings.filter[i].type = "select";
			    }
			}
		    }
		    
		    // create filter element
		    if (renderer.settings.filter[i]) {
			if (! renderer.settings.filter[i].searchword) {
			    renderer.settings.filter[i].searchword = "";
			}
			var filter_elem;
			if (renderer.settings.filter[i].type == "text") {
			    
			    var filter_text  = document.createElement("input");
			    filter_text.value = filter[i].searchword;
			    filter_text.setAttribute("style", "float: left; width: 100px; display: none;");
			    filter_text.i = i;
			    filter_text.onkeypress = function (e) {
				e = e || window.event;
				if (e.keyCode == 13) {
				    Retina.RendererInstances.table[index].settings.filter[this.i].searchword = this.value;
				    if (typeof renderer.settings.navigation_callback == "function") {
					var query = [];
					for (x in Retina.RendererInstances.table[index].settings.filter) {
					    if (Retina.RendererInstances.table[index].settings.filter.hasOwnProperty(x)) {
						if (Retina.RendererInstances.table[index].settings.filter[x].searchword.length > 0) {
						    query.push( { "searchword": Retina.RendererInstances.table[index].settings.filter[x].searchword, "field": Retina.RendererInstances.table[index].settings.header[x], "comparison": Retina.RendererInstances.table[index].settings.filter[x].operator || "=" } );
						}
					    }
					}
					renderer.settings.navigation_callback( { "query": query } );
				    } else {
					Retina.RendererInstances.table[index].settings.filter_changed = true;
					Retina.RendererInstances.table[index].render();
				    }
				}
			    };
			    
			    if (renderer.settings.filter[i].operator) {
				filter_elem = document.createElement("div");
				filter_elem.setAttribute("style", "float: left; margin-bottom: 0px; display: none;");
				filter_elem.className = "input-prepend";
				var operator_span = document.createElement("span");
				operator_span.setAttribute("style", "cursor: pointer;");
				operator_span.i = i;
				operator_span.onclick = function () {
				    for (var x=0; x< this.childNodes.length; x++) {
					if (this.childNodes[x].style.display == "") {
					    this.childNodes[x].style.display = "none";
					    if (x == this.childNodes.length - 1) {
						this.childNodes[0].style.display = "";
						Retina.RendererInstances.table[index].settings.filter[this.i].active_operator = 0;
					    } else {
						this.childNodes[x + 1].style.display = "";
						x++;
						Retina.RendererInstances.table[index].settings.filter[this.i].active_operator = x;
					    }
					}
				    }
				}
				operator_span.className = "add-on";
				for (var h=0; h<renderer.settings.filter[i].operator.length; h++) {
				    var operator = document.createElement("span");
				    operator.innerHTML = renderer.settings.filter[i].operator[h];
				    if (h==renderer.settings.filter[i].active_operator) {
					operator.setAttribute("style", "font-weight: bold;");
				    } else {
					operator.setAttribute("style", "display: none; font-weight: bold;");
				    }
				    operator.setAttribute("title", "click to switch filter operator");
				    operator_span.appendChild(operator);
				}
				filter_text.setAttribute("style", "position: relative; left: -3px; width: 80px;");
				filter_elem.appendChild(operator_span);
				filter_elem.appendChild(filter_text);
			    } else {
				filter_elem = filter_text;
			    }
			    
			} else if (renderer.settings.filter[i].type == "select") {
			    filter_elem = document.createElement("select");
			    filter_elem.setAttribute("style", "float: left; width: 100px; display: none;");
			    filter_elem.add(new Option("-show all-", ""), null);
			    var selopts = [];
			    for (h=0;h<tdata.length;h++) {
				if (tdata[h][header[i]].length) {
				    selopts[tdata[h][header[i]]] = 1;
				}
			    }
			    for (h in selopts) {
				if (h == renderer.settings.filter[i].searchword) {
				    filter_elem.add(new Option(h,h, true), null);
				} else {
				    filter_elem.add(new Option(h,h), null);
				}
			    }
			    filter_elem.i = i;
			    filter_elem.onchange = function () {
				Retina.RendererInstances.table[index].settings.filter[this.i].searchword = this.options[this.selectedIndex].value;
				Retina.RendererInstances.table[index].settings.filter_changed = true;
				Retina.RendererInstances.table[index].render();
			    }
			}
		    }
		    
		    // build header cell
		    var caret = document.createElement("table");
		    caret.setAttribute("style", "float: right");
		    var caret_tr1 = document.createElement("tr");
		    var caret_td1 = document.createElement("td");
		    caret_td1.setAttribute("style", "background-color: #CCC; padding: 0px 2px; line-height: 0px; -moz-border-radius: 4px; border-left: none;");
		    var caret_tr2 = document.createElement("tr");
		    var caret_td2 = document.createElement("td");
		    caret_td2.setAttribute("style", "background-color: #CCC; padding: 0px 2px; line-height: 0px; -moz-border-radius: 4px; border-left: none;");
		    caret_td1.appendChild(desc);
		    caret_td2.appendChild(asc);
		    caret_tr1.appendChild(caret_td1);
		    caret_tr2.appendChild(caret_td2);
		    caret.appendChild(caret_tr1);
		    caret.appendChild(caret_tr2);
		    var th = document.createElement("th");
		    var mw = 153;
		    if (renderer.settings.minwidths && renderer.settings.minwidths[i]) {
			mw = renderer.settings.minwidths[i];
		    }
		    th.setAttribute("style", "padding: 0px; padding-left: 4px; min-width: "+mw+"px;");
		    var th_div = document.createElement("div");
		    th_div.setAttribute("style", "float: left; position: relative; top: 4px;");
		    th_div.innerHTML = header[i];
		    th.appendChild(th_div);
		    if (renderer.settings.disable_sort[i]) {
			th_div.style.top = "-6px";
		    } else {
			th.appendChild(caret);
		    }
		    if (filter[i]) {
			var filter_icon = document.createElement("i");
			filter_icon.className = "icon-search";
			var is_active = "";
			if (filter[i].searchword) {
			    is_active = " border: 1px solid blue;";
			    filter_icon.setAttribute("title", "filtered for: '"+filter[i].searchword+"'");
			}
			filter_icon.setAttribute("style", "float: right; margin-top: 6px; cursor: pointer; margin-right: 2px;"+is_active);
			filter_icon.onclick = function () {
			    if (this.nextSibling.style.display == "") {
				this.nextSibling.style.display = "none";
				this.previousSibling.previousSibling.style.display = "";
			    } else {
				this.nextSibling.style.display = "";
				this.previousSibling.previousSibling.style.display = "none";
			    }
			}			  
			th.appendChild(filter_icon);
			th.appendChild(filter_elem);
		    }
		    tr.appendChild(th);
		}
	    }
	    thead.appendChild(tr);
	    table_element.appendChild(thead);
	    var tinner_elem = document.createElement("tbody");
	    
	    // check if the data is sorted, otherwise sort now
	    var disp;
	    if (renderer.settings.sorted) {
		disp = tdata;
	    } else {
		disp = tdata.sort(function (a,b) {		      
		    if (sortdir == 'desc') {
			var c = a; a=b; b=c;
		    }
		    if (sorttype[sortcol]) {
			switch (sorttype[sortcol]) {
			case "number":
			    if (typeof(a[header[sortcol]].replace) != 'function') {
				if (a[header[sortcol]]==b[header[sortcol]]) return 0;
				if (a[header[sortcol]]<b[header[sortcol]]) return -1;
			    } else {
				if (parseFloat(a[header[sortcol]].replace(/<(.|\n)*?>/g, ""))==parseFloat(b[header[sortcol]].replace(/<(.|\n)*?>/g, ""))) return 0;
				if (parseFloat(a[header[sortcol]].replace(/<(.|\n)*?>/g, ""))<parseFloat(b[header[sortcol]].replace(/<(.|\n)*?>/g, ""))) return -1;
			    }
			    return 1;
			    break;
			case "string":
			    if (a[header[sortcol]].replace(/<(.|\n)*?>/g, "")==b[header[sortcol]].replace(/<(.|\n)*?>/g, "")) return 0;
			    if (a[header[sortcol]].replace(/<(.|\n)*?>/g, "")<b[header[sortcol]].replace(/<(.|\n)*?>/g, "")) return -1;
			    return 1;
			    break;
			}
		    } else {
			if (a[header[sortcol]].replace(/<(.|\n)*?>/g, "")==b[header[sortcol]].replace(/<(.|\n)*?>/g, "")) return 0;
			if (a[header[sortcol]].replace(/<(.|\n)*?>/g, "")<b[header[sortcol]].replace(/<(.|\n)*?>/g, "")) return -1;
			return 1;
		    }
		});
		renderer.settings.sorted = true;
	    }
	    
	    // select the part of the data that will be displayed
	    if (renderer.settings.synchronous) {
		disp = disp.slice(offset, offset+rows);
	    }
	    
	    // create the table rows
	    for (i=0;i<disp.length;i++) {
		var tinner_row = document.createElement("tr");
		for (h=0; h<header.length; h++) {
		    if (! renderer.settings.invisible_columns[h]) {
			var tinner_cell = document.createElement("td");
			tinner_cell.innerHTML = disp[i][header[h]];
			if (renderer.settings.editable[h]) {
			    tinner_cell.addEventListener('click', function(e) {
				e = e || window.event;
				var ot = e.originalTarget || e.srcElement;
				var clicked_row_index;
				var clicked_cell_index;
				for (var x=0;x<ot.parentNode.children.length;x++) {
				    if (ot.parentNode.children[x] == ot) {
					clicked_cell_index = x;
				    }				      
				}
				for (var y=0;y<ot.parentNode.parentNode.children.length;y++) {
				    if (ot.parentNode.parentNode.children[y] == ot.parentNode) {
					clicked_row_index = y + offset;
					break;
				    }
				}
				
				var edit = document.createElement('input');
				edit.setAttribute('type', 'text');
				edit.setAttribute('value', Retina.RendererInstances.table[index].settings.tdata[clicked_row_index][header[clicked_cell_index]]);
				edit.addEventListener('keypress', function(e) {
				    e = e || window.event;
				    if (e.keyCode == 13) {
					Retina.RendererInstances.table[index].settings.tdata[clicked_row_index][header[clicked_cell_index]] = edit.value;
					if (Retina.RendererInstances.table[index].settings.edit_callback && typeof(Retina.RendererInstances.table[index].settings.edit_callback) == 'function') {
					    Retina.RendererInstances.table[index].settings.edit_callback.call(Retina.RendererInstances.table[index].settings.tdata);
					}
					Retina.RendererInstances.table[index].render();
				    }
				});
				edit.addEventListener('blur', function() {
				    Retina.RendererInstances.table[index].render();
				});
				ot.innerHTML = "";
				ot.appendChild(edit);
				edit.focus();
				if (typeof edit.selectionStart == "number") {
				    edit.selectionStart = 0;
				    edit.selectionEnd = edit.value.length;
				} else if (typeof document.selection != "undefined") {
				    document.selection.createRange().text = edit.value;
				}
			    });
			}
			tinner_row.appendChild(tinner_cell);
		    }
		}
		tinner_elem.appendChild(tinner_row);
	    }
	    
	    // render the table
	    table_element.appendChild(tinner_elem);
	    
	    // create the navigation
	    // first, previous
	    var prev_td = document.createElement("td");
	    prev_td.setAttribute("style", "text-align: left; width: 45px;");
	    prev_td.innerHTML = "&nbsp;";
	    if (offset > 0) {
		var first = document.createElement("i");
		first.setAttribute("class", "icon-fast-backward");
		first.setAttribute("title", "first");
		first.setAttribute("style", "cursor: pointer;");
		first.onclick = typeof renderer.settings.navigation_callback == "function" ? function () { renderer.settings.navigation_callback('first') } : function () {
		    Retina.RendererInstances.table[index].settings.offset = 0;
		    Retina.RendererInstances.table[index].render();
		}
		var prev = document.createElement("i");
		prev.setAttribute("class", "icon-step-backward");
		prev.setAttribute("title", "previous");
		prev.setAttribute("style", "cursor: pointer;");
		prev.onclick = typeof renderer.settings.navigation_callback == "function" ? function () { renderer.settings.navigation_callback('previous') } : function () {
		    Retina.RendererInstances.table[index].settings.offset -= rows;
		    if (Retina.RendererInstances.table[index].settings.offset < 0) {
			Retina.RendererInstances.table[index].settings.offset = 0;
		    }
		    Retina.RendererInstances.table[index].render();
		}
		prev_td.appendChild(first);
		prev_td.appendChild(prev);
	    }
	    
	    // next, last
	    var next_td = document.createElement("td");
	    next_td.setAttribute("style", "text-align: right; width: 45px;");
	    next_td.innerHTML = "&nbsp;";
	    if (offset + rows < (renderer.settings.numrows || tdata.length)) {
		var last = document.createElement("i");
		last.setAttribute("class", "icon-fast-forward");
		last.setAttribute("title", "last");
		last.setAttribute("style", "cursor: pointer;");
		last.onclick = typeof renderer.settings.navigation_callback == "function" ? function () { renderer.settings.navigation_callback('last') } : function () {
		    Retina.RendererInstances.table[index].settings.offset = tdata.length - rows;
		    if (Retina.RendererInstances.table[index].settings.offset < 0) {
			Retina.RendererInstances.table[index].settings.offset = 0;
		    }
		    Retina.RendererInstances.table[index].render();
		}
		var next = document.createElement("i");
		next.setAttribute("class", "icon-step-forward");
		next.setAttribute("title", "next");
		next.setAttribute("style", "cursor: pointer;");
		next.onclick = typeof renderer.settings.navigation_callback == "function" ? function () { renderer.settings.navigation_callback('next') } : function () {
		    Retina.RendererInstances.table[index].settings.offset += rows;
		    if (Retina.RendererInstances.table[index].settings.offset > tdata.length - 1) {
			Retina.RendererInstances.table[index].settings.offset = tdata.length - rows;
			if (Retina.RendererInstances.table[index].settings.offset < 0) {
			    Retina.RendererInstances.table[index].settings.offset = 0;
			}
		    }
		    Retina.RendererInstances.table[index].render();
		}
		next_td.appendChild(next);
		next_td.appendChild(last);
	    }
	    
	    // display of window offset
	    var showing = document.createElement("td");
	    showing.setAttribute("style", "text-align: center;");	  
	    showing.innerHTML = "showing rows "+ ((renderer.settings.offset || offset) + 1) +"-"+(disp.length + (renderer.settings.offset || offset))+" of "+(renderer.settings.numrows || tdata.length);
	    
	    // create the table to host navigation
	    var bottom_table = document.createElement("table");
	    bottom_table.setAttribute("style", "width: 100%");
	    var bottom_row = document.createElement("tr");
	    bottom_row.appendChild(prev_td);
	    bottom_row.appendChild(showing);
	    bottom_row.appendChild(next_td);
	    bottom_table.appendChild(bottom_row);
	    
	    // goto
	    var goto_label = document.createElement("span");
	    goto_label.innerHTML = "goto row ";
	    var goto_text = document.createElement("input");
	    goto_text.setAttribute("value", offset + 1);
	    goto_text.setAttribute("class", "span1");
	    goto_text.setAttribute("style", "position: relative; top: 4px; height: 16px;");
	    goto_text.onkeypress = function (e) {
		e = e || window.event;
		if (e.keyCode == 13) {
		    if (typeof renderer.settings.navigation_callback == "function") {
			renderer.settings.navigation_callback({'goto': parseInt(this.value) - 1 });
		    } else {
			Retina.RendererInstances.table[index].settings.offset = parseInt(this.value) - 1;
			if (Retina.RendererInstances.table[index].settings.offset < 0) {
			    Retina.RendererInstances.table[index].settings.offset = 0;
			}
			if (Retina.RendererInstances.table[index].settings.offset > rows) {
			    Retina.RendererInstances.table[index].settings.offset = rows;
			}
			Retina.RendererInstances.table[index].render();
		    }
		}
	    };
	    
	    // clear filter button
	    var clear_btn = document.createElement("input");
	    clear_btn.setAttribute("type", "button");
	    clear_btn.setAttribute("class", "btn");
	    clear_btn.setAttribute("value", "clear all filters");
	    clear_btn.style.marginLeft = "10px";
	    clear_btn.onclick = function () {
		    for (i in Retina.RendererInstances.table[index].settings.filter) {
		        Retina.RendererInstances.table[index].settings.filter[i].searchword = "";
		    }
		    if (typeof renderer.settings.navigation_callback == "function") {
		        renderer.settings.navigation_callback({"goto": 0, "query": "default", "sort": "default"});
	        } else {
		        Retina.RendererInstances.table[index].settings.sorted = false;
		        Retina.RendererInstances.table[index].render();
	        }
	    };
	    
	    // rows per page
	    var perpage = document.createElement("input");
	    perpage.setAttribute("type", "text");
	    perpage.setAttribute("value", rows);
	    perpage.setAttribute("style", "position: relative; top: 4px; height: 16px; width: 30px;");
	    perpage.onkeypress = function (e) {
		e = e || window.event;
		if (e.keyCode == 13) {
		    if (typeof renderer.settings.navigation_callback == "function") {
			renderer.settings.navigation_callback({'limit': parseInt(this.value) });
		    } else {
			Retina.RendererInstances.table[index].settings.offset = 0;
			Retina.RendererInstances.table[index].settings.rows_per_page = parseInt(this.value);
			Retina.RendererInstances.table[index].render();
		    }
		}
	    };
	    var ppspan1 = document.createElement("span");
	    ppspan1.style.marginLeft = "10px";
	    ppspan1.innerHTML = " show ";
	    var ppspan2 = document.createElement("span");
	    ppspan2.style.marginRight = "10px";
	    ppspan2.innerHTML = " rows at a time";
	    
	    // handle onclick event
	    if (renderer.settings.onclick) {
		table_element.onclick = function (e) {
		    e = e || window.event;
		    var ot = e.originalTarget || e.srcElement;
		    if (ot.nodeName == "TD") {
			var clicked_row = [];
			var clicked_row_index;
			var clicked_cell_index;
			for (var x=0;x<ot.parentNode.children.length;x++) {
			    if (ot.parentNode.children[x] == ot) {
				clicked_cell_index = x;
			    }
			    clicked_row.push(ot.parentNode.children[x].innerHTML);
			}
			for (var y=0;y<ot.parentNode.parentNode.children.length;y++) {
			    if (ot.parentNode.parentNode.children[y] == ot.parentNode) {
				clicked_row_index = y + offset;
				break;
			    }
			}
			var clicked_cell = ot.innerHTML;
			Retina.RendererInstances.table[index].settings.onclick(clicked_row, clicked_cell, clicked_row_index, clicked_cell_index);
		    }
		};
	    }
	    
	    var col_sel_span = document.createElement("span");
	    var col_sel_btn = document.createElement("input");
	    col_sel_btn.setAttribute("class", "btn");
	    col_sel_btn.setAttribute("type", "button");
	    col_sel_btn.setAttribute("value", "select columns");
	    var col_sel = document.createElement("div");
	    col_sel.setAttribute('style', "position: absolute; top: 5px; left: 570px; min-width: 150px; border: 1px solid #BBB; background-color: white; z-index: 1000; display: none; box-shadow: 4px 4px 4px #666; padding: 2px;");
	    col_sel_btn.addEventListener("click", function () {
		if (col_sel.style.display == "none") {
		    col_sel.style.display = "";
		} else {
		    col_sel.style.display = "none";
		}
	    });
	    var colsel_html = "<input type='button' class='btn btn-mini' style='float: right;' value='OK' onclick='Retina.RendererInstances.table["+index+"].update_visible_columns("+index+");'><table id='table_colsel_table_"+index+"'>";
	    for (ii=0;ii<renderer.settings.header.length;ii++) {
		var checked = " checked";
		if (renderer.settings.invisible_columns[ii]) {
		    checked = "";
		}
		colsel_html += "<tr><td><input style='margin-right: 5px;' type='checkbox'"+checked+"></td><td>"+renderer.settings.header[ii]+"</td></tr>";
	    }
	    colsel_html += "</table>";
	    col_sel.innerHTML = colsel_html;
	    col_sel_span.appendChild(col_sel_btn);
	    col_sel_span.appendChild(col_sel);
	    
	    var options_icon = document.createElement("div");
	    options_icon.innerHTML = "<i class='icon-cog'></i>";
	    options_icon.title ='table options, click to show';
	    options_icon.className = "btn";
	    options_icon.setAttribute("style", "cursor: pointer; position: relative; top: -1px; margin-bottom: 7px;");
	    options_icon.onclick = function () {
		this.nextSibling.style.display = "";
		this.style.display = "none";
	    }
	    var options_span = document.createElement("div");
	    options_span.setAttribute('style', "display: none; position: relative; top: -5px;");
	    options_span.innerHTML = "<div title='close options' onclick='this.parentNode.previousSibling.style.display=\"\";this.parentNode.style.display=\"none\";' style='cursor: pointer; margin-right: 5px;' class='btn'><i class='icon-remove'></div>";
	    
	    // append navigation to target element
	    if (renderer.settings.hide_options == false) {
		target.appendChild(options_icon);
		target.appendChild(options_span);
		options_span.appendChild(goto_label);
		options_span.appendChild(goto_text);
		options_span.appendChild(clear_btn);
		options_span.appendChild(ppspan1);
		options_span.appendChild(perpage);
		options_span.appendChild(ppspan2);
		options_span.appendChild(col_sel_span);
	    }
	    target.appendChild(table_element);
	    target.appendChild(bottom_table);	  
	    
	    return renderer;
	}
    });
}).call(this);
