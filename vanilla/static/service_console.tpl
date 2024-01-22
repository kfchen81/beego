<!DOCTYPE html>
<html>
	<head>
		<meta http-equiv="Content-Type" content="text/html; charset=UTF-8">
		<title>{{.ServiceName}}  Service Console - Golang</title>

		<link type="text/css" rel="stylesheet" media="all" href="/static/lib/bootstrap-3.3.4/css/bootstrap.min.css">
		<link type="text/css" rel="stylesheet" media="all" href="/static/css/base.css">
		<link type="text/css" rel="stylesheet" media="all" href="/static/css/apiserver.css">
		<link type="text/css" rel="stylesheet" href="/static/lib/jsoneditor/jsoneditor.min.css">
		<link type="text/css" rel="stylesheet" href="/static/lib/codemirror-5.7/lib/codemirror.css">
		<link type="text/css" rel="stylesheet" href="/static/lib/codemirror-5.7/theme/monokai.css">
		<link type="text/css" rel="stylesheet" href="/static/lib/codemirror-5.7/addon/scroll/simplescrollbars.css">
	</head>
	<body>
		<nav class="navbar navbar-default" style="margin-bottom: 5px;">
			<div class="container-fluid">
				<div class="navbar-header">
					<a class="navbar-brand" href="#">{{.ServiceName}} Service Console - Golang</a>
				</div>
			</div><!-- /.container-fluid -->
		</nav>

		<form class="p15 clearfix" style="margin-top:5px; padding-top:5px; background-color:white; border:solid 1px #CFCFCF; margin:20px;">
			<div class="fl" style="width:300px;">
				<label for="resource">选择资源</label>
				<select id="resource" class="form-control" style="width:200px; display:inline-block;">
					{{range .Resources}}
					<option value="{{.}}">{{.}}</option>
					{{end}}
				</select>
			</div>
			<div class="fl" style="width:560px;">
				<label for="resource" style="vertical-align:top;">数据</label>
				<div class="xa-data xui-data" style="width:500px; height:200px; display:inline-block;"></div>
			</div>
			<div class="fl" style="width:100px;">
				<a class="btn btn-success xa-link" style="display:block;" data-action="get">GET</a>
				<a class="btn btn-success mt10 xa-link" style="display:block;" data-action="put">PUT</a>
				<a class="btn btn-success mt10 xa-link" style="display:block;" data-action="post">POST</a>
				<a class="btn btn-danger mt10 xa-link" style="display:block;" data-action="delete">DELETE</a>
			</div>
		</form>

		<div class="clearfix">
			<div id="result" class="fl" style="height:800px; width:50%;"></div>
			<div id="queries" class="fl ml10" style="width:48%;">
				<table class="table table-bordered">
					<thead>
						<th width="40px">源</th>
						<th width="85%">SQL查询<span class="xa-sqlCount"></span></th>
						<th>时间</th>
					</thead>
					<tbody class="xa-table">
					</tbody>
				</table>
			</div>
		</div>

		<!-- 3rd party lib -->
		<script type="text/javascript" src="/static/lib/jquery/jquery-1.11.2.min.js"></script>
		<script type="text/javascript" src="/static/lib/underscore-1.7.0.min.js"></script>
		<script type="text/javascript" src="/static/lib/jsoneditor/jsoneditor.min.js"></script>
		<script type="text/javascript" src="/static/lib/codemirror-5.7/lib/codemirror.js"></script>
		<script type="text/javascript" src="/static/lib/codemirror-5.7/mode/javascript/javascript.js"></script>
		<script type="text/javascript" src="/static/lib/codemirror-5.7/addon/scroll/simplescrollbars.js"></script>

		<script type="text/javascript">
		var dataCodeMirror = null;
		var resultViewer = null;

		var __createCodeEditor = function(selector, mode, value) {
			var $code = this.$(selector);
			var codeMirror = CodeMirror($code.get(0), {
				value: value,
				mode: mode,
				lineNumbers: true,
				theme: 'monokai',
				scrollbarStyle: 'simple',
				lineWrapping: true
			});
			$code.find('.CodeMirror').height($code.outerHeight() - 32);

			return codeMirror;
		};

		var __createResultViewer = function() {
			var container = document.getElementById('result');
			var options = {
				mode: 'text'
			};

			var editor = new JSONEditor(container, options, '');
			return editor;
		}

		var __displayQueries = function(queries) {
			var buf = [];
			var index = 0;
			queries.forEach(function(query) {
				index += 1;
				buf.push('<tr style="cursor:pointer;" class="xa-span" data-target="'+index+'">');
				buf.push('<td>'+query.source+'</td>');
				buf.push('<td>'+query.query+'</td>');
				buf.push('<td>'+query.time+'</td>');
				buf.push('</tr>');
				buf.push('<tr style="display:none;" data-index="'+index+'">');
				buf.push('<td colspan="3" style="background-color:white;">'+query.stack+'</td>');
				buf.push('</tr>');
			});

			$('.xa-table').html(buf.join('\n'));
			$('.xa-sqlCount').text('(' + queries.length + ')');
		}

		$(document).ready(function() {
			dataCodeMirror = __createCodeEditor('.xa-data', 'javascript', "data = {\n  id:1,\n}");
			resultViewer = __createResultViewer();

			var failCallback = function() {
				alert('访问失败！请查看日志');
			}

			$(document).delegate('.xa-span', 'click', function(event) {
				var $tr = $(event.currentTarget);
				var $targetTr = $('[data-index="'+$tr.data('target')+'"]');
				if ($targetTr.is(':visible')) {
					$targetTr.hide();
				} else {
					$targetTr.show();
				}
			});

			$('.xa-link').click(function(event) {
				var $link = $(event.currentTarget);
				var action = $link.data('action');
				var data = dataCodeMirror.getValue();
				data = eval(data.replace(/'/g, '\''));
				var resource = $('#resource').val();
				var pos = resource.lastIndexOf('.');
				resource = '/' + resource.substring(0, pos) + '/api/' + resource.substring(pos+1) + '/';
				var url = resource.replace(/\./g, '/');

				resultViewer.set('fetching...');
				__displayQueries([]);

				if (action == 'get') {
					$.get(url, data, function(data) {
						var queries = data.queries || [];
						delete data.queries;
						__displayQueries(queries);

						if (data.code !== 200) {
							var errMsg = data.errMsg;
							if (typeof errMsg === 'object') {
								errMsg = JSON.stringify(errMsg);
							}

							var innerErrMsg = data.innerErrMsg;
							if (typeof innerErrMsg === 'object') {
								innerErrMsg = JSON.stringify(innerErrMsg);
							}

							var buffer = ['Error: ' + errMsg, '\n', innerErrMsg];
							$('.jsoneditor textarea').val(buffer.join('\n'));
						} else {
							resultViewer.set(data);
						}
					}).fail(failCallback);
				} else if (action == 'put' || action == 'post' || action == 'delete') {
					if (action !== 'post') {
						url += '?_method=' + action;
					}
					$.post(url, data, function(data) {
						var queries = data.queries || [];
						delete data.queries;
						__displayQueries(queries);

						if (data.code !== 200) {
							var errMsg = data.errMsg;
							if (typeof errMsg === 'object') {
								errMsg = JSON.stringify(errMsg);
							}

							var innerErrMsg = data.innerErrMsg;
							if (typeof innerErrMsg === 'object') {
								innerErrMsg = JSON.stringify(innerErrMsg);
							}

							var buffer = ['Error: ' + errMsg, '\n', innerErrMsg];
							$('.jsoneditor textarea').val(buffer.join('\n'));
						} else {
							resultViewer.set(data);
						}
					}).fail(failCallback);
				}
			});
		})
		</script>
	</body>
</html>
