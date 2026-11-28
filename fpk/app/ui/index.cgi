#!/bin/bash
# GoSvg — fnOS CGI documentation page (desktop "Open" entry).
# Served by fnOS nginx at /cgi/ThirdParty/gosvg/index.cgi/
#
# 文档正文已渲染为静态 HTML，内嵌在下方 BODY 变量中，不依赖网络或 python markdown。
# 需要修改文档时，编辑 BODY 里的 HTML 即可（或本地重新生成后替换）。


echo "Content-Type: text/html; charset=utf-8"
echo "Cache-Control: no-store"
echo ""

BODY=$(cat <<'HTML_BODY_EOF'
<blockquote>
<p>纯 Go 实现的通用 SVG 渲染命令行工具集 · v0.5.0</p>
<p>项目地址：<a href="https://github.com/Blue-Mink/gosvg">github.com/Blue-Mink/gosvg</a></p>
</blockquote>
<h2 id="_1">简介与准备</h2>
<p>GoSvg 是一套用 Go 编写的 SVG 渲染命令行工具。飞牛应用把 4 个工具安装到 NAS 上，通过 SSH 登录后即可在命令行使用，没有网页操作界面。</p>
<p>工具统一位于 <code>/vol1/@appcenter/gosvg/bin/</code>。示例前先设一个变量简化输入：</p>
<pre><code class="language-sh"># 设为工具目录（后面 $B 即代表它）
B=/vol1/@appcenter/gosvg/bin

# 准备一个示例 SVG（demo.svg）
cat &gt; demo.svg &lt;&lt;'EOF'
&lt;svg xmlns=&quot;http://www.w3.org/2000/svg&quot; viewBox=&quot;0 0 200 120&quot;&gt;
  &lt;rect width=&quot;200&quot; height=&quot;120&quot; fill=&quot;#eef3ff&quot;/&gt;
  &lt;circle cx=&quot;60&quot; cy=&quot;60&quot; r=&quot;40&quot; fill=&quot;#2f9bff&quot;/&gt;
  &lt;rect x=&quot;110&quot; y=&quot;30&quot; width=&quot;60&quot; height=&quot;60&quot; rx=&quot;10&quot; fill=&quot;#ff6b8a&quot;/&gt;
  &lt;text x=&quot;100&quot; y=&quot;112&quot; font-size=&quot;16&quot; text-anchor=&quot;middle&quot; fill=&quot;#15357a&quot;&gt;GoSvg&lt;/text&gt;
&lt;/svg&gt;
EOF
</code></pre>
<p>下文的示例都基于这个 <code>demo.svg</code>，你可以直接复制照做。参数以各工具的 <code>-h</code> 输出为准。</p>
<h2 id="1-gosvg-render">1. gosvg-render — 单文件渲染</h2>
<p>把一个 SVG 文件渲染成 PNG 或 JPEG 位图。输入/输出是<strong>位置参数</strong>（输入在前、输出在后），也可以用 <code>-</code> 走标准输入输出。</p>
<h3 id="_2">参数表</h3>
<table>
<thead>
<tr>
<th style="text-align: left;">参数</th>
<th style="text-align: left;">说明</th>
</tr>
</thead>
<tbody>
<tr>
<td style="text-align: left;"><code>-size N</code></td>
<td style="text-align: left;">宽高相同，默认 512；被 <code>-W</code>/<code>-H</code> 覆盖</td>
</tr>
<tr>
<td style="text-align: left;"><code>-W N</code> / <code>-H N</code></td>
<td style="text-align: left;">分别指定输出宽、高（0 = 用 -size）</td>
</tr>
<tr>
<td style="text-align: left;"><code>-bg #RGB / #RRGGBB</code></td>
<td style="text-align: left;">不透明底色，默认透明；JPEG 透明区默认合成白色</td>
</tr>
<tr>
<td style="text-align: left;"><code>-format png / jpeg</code></td>
<td style="text-align: left;">输出格式；省略则按扩展名推断（.jpg/.jpeg→jpeg，其余→png）</td>
</tr>
<tr>
<td style="text-align: left;"><code>-quality N</code></td>
<td style="text-align: left;">JPEG 质量 1–100，默认 90</td>
</tr>
<tr>
<td style="text-align: left;"><code>-strict</code></td>
<td style="text-align: left;">渲染期出现静默跳过（未识别元素、缺失引用等）则退出码 3</td>
</tr>
<tr>
<td style="text-align: left;"><code>-</code></td>
<td style="text-align: left;">输入为 <code>-</code> 时从 stdin 读；省略输出路径时写到 stdout</td>
</tr>
</tbody>
</table>
<h3 id="a256256-png">实例 A：基础渲染（256×256 PNG）</h3>
<pre><code class="language-sh">$ $B/gosvg-render -size 256 demo.svg out.png
# 退出码 0，生成 out.png（6023 字节，PNG 尺寸 256×256）
</code></pre>
<h3 id="b-jpeg">实例 B：指定宽高 + 白底 JPEG</h3>
<pre><code class="language-sh">$ $B/gosvg-render -W 400 -H 240 -bg '#ffffff' -format jpeg -quality 85 demo.svg out.jpg
# 退出码 0，生成 out.jpg（7300 字节）
</code></pre>
<h3 id="cstdin-stdout">实例 C：stdin → stdout（管道）</h3>
<pre><code class="language-sh">$ cat demo.svg | $B/gosvg-render -size 128 - &gt; stdin.png
# 退出码 0，stdin.png 2484 字节
</code></pre>
<h3 id="d">实例 D：严格模式</h3>
<pre><code class="language-sh">$ $B/gosvg-render -strict -size 64 demo.svg strict.png
# 正常 SVG：退出码 0；若有被跳过的元素则退出 3 并打印诊断
</code></pre>
<blockquote>
<p>[!TIP]
旧写法 <code>gosvg-render in.svg out.png 256</code> 仍兼容；但位置 size 与 <code>-size</code>/<code>-W</code>/<code>-H</code> 同时出现会报错退出 2。</p>
</blockquote>
<h2 id="2-gosvg-corpus">2. gosvg-corpus — 批量语料渲染</h2>
<p>对一个目录里的所有 SVG 批量渲染，输出统计报告。目录作为位置参数传入。</p>
<h3 id="_3">参数表</h3>
<table>
<thead>
<tr>
<th style="text-align: left;">参数</th>
<th style="text-align: left;">说明</th>
</tr>
</thead>
<tbody>
<tr>
<td style="text-align: left;"><code>-size N</code></td>
<td style="text-align: left;">渲染边长（正方形），默认 64</td>
</tr>
<tr>
<td style="text-align: left;"><code>-json</code></td>
<td style="text-align: left;">以 JSON 输出报告（默认是人类可读文本）</td>
</tr>
</tbody>
</table>
<h3 id="2-svg">实例：批量渲染 2 个 SVG</h3>
<pre><code class="language-sh">$ mkdir svgs &amp;&amp; cp demo.svg svgs/a.svg &amp;&amp; cp demo.svg svgs/b.svg
$ $B/gosvg-corpus -size 64 svgs
语料目录: svgs
SVG 文件数: 2   渲染尺寸: 64x64   耗时: 6ms
渲染出内容: 2 (100.0%)
渲染全透明: 0 (0.0%)
Parse 失败: 0 (0.0%)
Render 失败: 0

# 加 -json 得到结构化报告，便于脚本处理
$ $B/gosvg-corpus -size 64 -json svgs
{ &quot;dirs&quot;:[&quot;svgs&quot;], &quot;size&quot;:64, &quot;total&quot;:2, &quot;with_content&quot;:2,
  &quot;blank&quot;:0, &quot;parse_failed&quot;:0, &quot;render_failed&quot;:0, &quot;elapsed_ms&quot;:6 }
</code></pre>
<h2 id="3-gosvg-iou">3. gosvg-iou — 渲染一致性对账</h2>
<p>把一个 SVG 目录渲染后，与参照图目录里的 PNG 逐文件计算 IoU（交并比）相似度，用于回归测试/渲染对账。SVG 文件名去掉扩展名后与参照 PNG 名对应（如 <code>a.svg</code> ↔ <code>a.png</code>）。</p>
<h3 id="_4">参数表</h3>
<table>
<thead>
<tr>
<th style="text-align: left;">参数</th>
<th style="text-align: left;">说明</th>
</tr>
</thead>
<tbody>
<tr>
<td style="text-align: left;"><code>-svg 目录</code></td>
<td style="text-align: left;">待渲染的 SVG 目录（与 -baseline 至少给一个）</td>
</tr>
<tr>
<td style="text-align: left;"><code>-ref 目录</code></td>
<td style="text-align: left;">参照图目录，内含 PNG（必填）</td>
</tr>
<tr>
<td style="text-align: left;"><code>-baseline 目录</code></td>
<td style="text-align: left;">基线渲染目录；给定则额外算与基线的 IoU 差异</td>
</tr>
<tr>
<td style="text-align: left;"><code>-size N</code></td>
<td style="text-align: left;">渲染边长，默认 256</td>
</tr>
<tr>
<td style="text-align: left;"><code>-alpha N</code></td>
<td style="text-align: left;">alpha 阈值；0=alpha&gt;0 视为不透明，8=报告口径</td>
</tr>
<tr>
<td style="text-align: left;"><code>-min F</code></td>
<td style="text-align: left;">只列出 IoU 低于 F 的文件（0=不限制）</td>
</tr>
<tr>
<td style="text-align: left;"><code>-out 目录</code></td>
<td style="text-align: left;">渲染输出目录；为空则不落盘</td>
</tr>
<tr>
<td style="text-align: left;"><code>-json 文件</code></td>
<td style="text-align: left;">把完整结果写成 JSON 到指定文件</td>
</tr>
<tr>
<td style="text-align: left;"><code>-quiet</code></td>
<td style="text-align: left;">只输出汇总，不逐文件打印</td>
</tr>
</tbody>
</table>
<h3 id="a">实例 A：正常对账（渲染图与参照一致）</h3>
<pre><code class="language-sh">$ mkdir refs &amp;&amp; cp out.png refs/a.png &amp;&amp; cp out.png refs/b.png
$ $B/gosvg-iou -size 256 -svg svgs -ref refs

alpha&gt;0  size=256  compared=2  missing=0
average IoU = 1.000000
IoU &lt; 0.5   = 0
</code></pre>
<h3 id="b">实例 B：制造差异 + 阈值过滤（找出退化项）</h3>
<pre><code class="language-sh"># 把 refs/b.png 换成一张明显不同的图后：
$ $B/gosvg-iou -size 256 -svg svgs -ref refs -min 0.9
LOW  b                                  iou=0.0000 ref=1 got=0

alpha&gt;0  size=256  compared=2  missing=0
average IoU = 0.500000
IoU &lt; 0.5   = 1
</code></pre>
<h3 id="c-json">实例 C：导出 JSON 报告</h3>
<pre><code class="language-sh">$ $B/gosvg-iou -size 256 -quiet -json iou.json -svg svgs -ref refs
# iou.json 含 compared / average_iou / samples[]（每个文件的 iou、像素数）
</code></pre>
<blockquote>
<p>[!TIP]
若只给 <code>-baseline</code> 不给 <code>-svg</code>，则只做两组已有图的比对；两者都省略会提示"至少需要 -svg 或 -baseline 之一"。</p>
</blockquote>
<h2 id="4-gosvg-verify">4. gosvg-verify — 项目验证工具</h2>
<p>对 Go 源码项目运行完整验证门禁——单元测试、竞态检测、交叉编译、fuzz 与 golden 基线比对。这是开发工具，日常使用其他三个渲染工具时不需要它；运行时需在与源码布局匹配的目录中执行。</p>
<pre><code class="language-sh">$ cd /path/to/gosvg
$ $B/gosvg-verify
# 依次执行各检查项并输出结果
</code></pre>
<h2 id="_5">常见问题</h2>
<h3 id="q-non-running">Q：应用中心显示 non-running，工具还能用吗？</h3>
<p>能。显示状态由轻量哨兵进程维持，仅用于满足 start/stop/status 生命周期，与命令行工具无关。只要 <code>bin/</code> 下二进制还在，SSH 调用照常可用。</p>
<h3 id="q-svg">Q：支持哪些 SVG 特性？</h3>
<p>支持常见元素（path/形状/text/渐变/mask/clipPath/marker/pattern/style 等）与 CSS 选择器级联，含 <code>vector-effect="non-scaling-stroke"</code>。明确不支持：filter、textPath、foreignObject、动画/脚本、CSS 伪类等（命中时静默跳过，可用 <code>-strict</code> 发现）。</p>
<h3 id="q">Q：中文文字渲染正常吗？</h3>
<p>内置字体可能不含中文字形，中文渲染有缺字属于已知限制。如需完整的中文支持，需要使用带中文字形的字体。</p>
HTML_BODY_EOF
)

cat <<HTML
<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>GoSvg 使用文档</title>
<style>
  :root{color-scheme:light}
  *{box-sizing:border-box}
  body{margin:0;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;background:#f5f7fb;color:#1f2733;line-height:1.75}
  .wrap{width:100%;max-width:1120px;margin:0 auto;padding:24px 22px 14px}
  .topbar{background:linear-gradient(135deg,#1b2a4a,#0e1830);border-radius:18px;padding:18px 24px;color:#dce6ff;margin-bottom:16px;display:flex;align-items:center;gap:18px;box-shadow:0 8px 24px rgba(20,40,90,.18)}
  .topbar h1{color:#fff;margin:0;font-size:25px}
  .topbar .sub{color:#9fb4d8;font-size:13.5px;margin-top:3px}
  .card{background:#fff;border:1px solid #e6eaf1;border-radius:16px;padding:6px 32px 26px;margin-bottom:12px;box-shadow:0 2px 10px rgba(20,40,90,.04)}
  h1{font-size:24px;color:#15357a;border-bottom:2px solid #eef1f7;padding-bottom:12px}
  h2{font-size:19px;margin:26px 0 10px;color:#15357a;border-bottom:1px solid #eef1f7;padding-bottom:8px}
  h3{font-size:15.5px;margin:20px 0 8px;color:#1f3f8a}
  p{margin:8px 0;font-size:14px;color:#3b4759}
  code,pre{font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
  :not(pre)>code{background:#eef1f7;padding:2px 6px;border-radius:5px;font-size:12.5px;color:#1b3a7a}
  pre{background:#0f1b33;color:#dce6ff;padding:14px 18px;border-radius:10px;font-size:12.5px;overflow-x:auto;margin:10px 0;line-height:1.7;-webkit-overflow-scrolling:touch}
  table{width:100%;border-collapse:collapse;font-size:13px;margin:10px 0;display:block;overflow-x:auto;-webkit-overflow-scrolling:touch}
  th,td{text-align:left;padding:8px 12px;border-bottom:1px solid #eef1f7;vertical-align:top;white-space:nowrap}
  th{color:#15357a;font-weight:600;background:#f7f9fd}
  ul{margin:8px 0;padding-left:20px;font-size:14px;color:#3b4759}
  li{margin:5px 0}
  blockquote{background:#eef3ff;border-left:4px solid #2a5fd6;border-radius:8px;padding:10px 16px;margin:14px 0;color:#2b4a86;font-size:13.5px}
  blockquote p{margin:4px 0;color:#2b4a86}
  .warn{background:#fff1f1;border-left:4px solid #d64550;padding:14px 18px;border-radius:8px;color:#8a2029;font-size:14px}
  a{color:#2a5fd6;text-decoration:none} a:hover{text-decoration:underline;word-break:break-all}
  .foot{text-align:center;color:#9aa4b2;font-size:12px;padding:16px 0 4px}

  /* 移动端适配 */
  @media (max-width:640px){
    .wrap{padding:12px 10px 8px}
    .topbar{padding:14px 16px;gap:12px;border-radius:14px;margin-bottom:10px}
    .topbar svg{width:44px;height:44px}
    .topbar h1{font-size:19px}
    .topbar .sub{font-size:12px;line-height:1.5}
    .card{padding:4px 16px 18px;border-radius:13px;margin-bottom:8px}
    h1{font-size:20px}
    h2{font-size:17px;margin:22px 0 8px}
    h3{font-size:14.5px;margin:16px 0 6px;line-height:1.5}
    p{font-size:13.5px;line-height:1.75}
    :not(pre)>code{font-size:12px;padding:1px 5px}
    pre{font-size:11.5px;padding:12px 14px;margin:8px -4px;border-radius:8px}
    table{font-size:12px;margin:8px -4px}
    th,td{padding:7px 10px}
    blockquote{font-size:12.5px;padding:9px 12px;margin:12px 0}
    .foot{padding:12px 0 4px;font-size:11px}
  }
</style>
</head>
<body>
<div class="wrap">

  <div class="topbar">
    <svg viewBox="0 0 512 512" width="58" height="58" xmlns="http://www.w3.org/2000/svg" style="border-radius:14px;flex:none"><defs><linearGradient id="codebg" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#1b2a4a"/><stop offset="1" stop-color="#0e1830"/></linearGradient></defs><rect width="512" height="512" rx="112" fill="url(#codebg)"/><rect x="76" y="140" width="360" height="232" rx="24" fill="#ffffff" fill-opacity=".04" stroke="#5ee7df" stroke-opacity=".25" stroke-width="3"/><g fill="#5ee7df"><circle cx="104" cy="170" r="9"/><circle cx="136" cy="170" r="9"/><circle cx="168" cy="170" r="9"/></g><g fill="none" stroke="#5ee7df" stroke-width="22" stroke-linecap="round" stroke-linejoin="round"><path d="M200 236 L 152 276 L 200 316"/><path d="M312 236 L 360 276 L 312 316"/></g><path d="M282 222 L 232 330" fill="none" stroke="#ff6b8a" stroke-width="20" stroke-linecap="round"/></svg>
    <div>
      <h1>GoSvg 使用文档</h1>
      <div class="sub">纯 Go 实现的通用 SVG 渲染命令行工具集 · v0.5.0</div>
    </div>
  </div>

  <div class="card">
$BODY
  </div>

  <div class="foot">GoSvg · SVG 渲染命令行工具集 · v0.5.0</div>
</div>

<script>
/* doc-v15: 让父 iframe 高度贴合内容, 消除满高 iframe 下 footer 下方的多余空白.
   仅在同源 iframe 内生效(window.frameElement 可取); 直接打开/跨域时静默跳过. */
(function(){
  function fit(){
    var me = window.frameElement;
    if(!me) return;                       // 非 iframe(直接打开/移动端 webview): 不动
    try{
      me.style.height = 'auto';           // 先释放, 才能量到真实内容高
      var h = document.documentElement.scrollHeight;
      me.style.height = h + 'px';         // 内联样式覆盖飞牛的 height:100%
      me.style.maxWidth = '100%';
    }catch(e){}                           // 跨域拿不到 frameElement: 退回现状, 不会更糟
  }
  fit();
  window.addEventListener('load', fit);
  setTimeout(fit, 150);
  setTimeout(fit, 600);
  if(window.ResizeObserver){
    var last = 0, guard = 0;
    try{
      new ResizeObserver(function(){
        if(guard++ > 30) return;          // 防止极端情况下反复触发
        var h = document.documentElement.scrollHeight;
        if(Math.abs(h - last) > 2){ last = h; fit(); }
      }).observe(document.body);
    }catch(e){}
  }
})();
</script>
</body>
</html>
HTML
