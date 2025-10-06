const express = require("express");
const os = require("os");
const path = require("path");
const app = express ();

// Serve static files from the img directory
app.use('/img', express.static(path.join(__dirname, 'img')));

// Environment Variable. This is configurable, it defaults to "abc123"
const value = process.env.DISPLAY_VALUE || "abc123";

app.get("/", (_req, res) => {
	res.set("Content-Type", "text/html");
	res.send(`<!doctype html>
<html>
  	<head>
  	  <meta charset="utf-8" />
  	  <title>Pulumi Demo App</title>
  	  <meta name="viewport" content="width=device-width, initial-scale=1" />
  	<style>
  		body { font-family: system-ui, -apple-system, Segoe UI, Roboto, sans-serif; margin: 40px; }
  		code { background: #f2f2f2; padding: 2px 6px; border-radius: 4px; }
  		.chip { display: inline-block; padding: 8px 12px; border-radius: 999px; border: 1px solid #ddd; }
  		.link { color: #0066cc; text-decoration: none; border-bottom: 1px solid #0066cc; }
  		.link:hover { color: #0052a3; border-bottom-color: #0052a3; }

  		.diagram-container {
  			margin: 20px 0;
  			text-align: center;
  		}
  		.diagram-preview {
  			max-width: 300px;
  			border: 2px solid #ddd;
  			border-radius: 8px;
  			cursor: pointer;
  			transition: transform 0.2s, box-shadow 0.2s;
  		}
  		.diagram-preview:hover {
  			transform: scale(1.02);
  			box-shadow: 0 4px 12px rgba(0,0,0,0.15);
  		}

  		.modal {
  			display: none;
  			position: fixed;
  			z-index: 1000;
  			left: 0;
  			top: 0;
  			width: 100%;
  			height: 100%;
  			background-color: rgba(0,0,0,0.8);
  			cursor: pointer;
  		}
  		.modal.active {
  			display: flex;
  			align-items: center;
  			justify-content: center;
  		}
  		.modal img {
  			max-width: 90%;
  			max-height: 90%;
  			border-radius: 8px;
  		}
  	</style>
  	</head>
  	<body>
  		<h1>Pulumi + Kubernetes = ❤️</h1>
  		<p>This page is serving a configurable value using a Pulumi environment variable.</p>
  		<p>Value: <span class="chip"><code>${value}</code></span></p>

  		<h2>Application Flow Diagram</h2>
  		<div class="diagram-container">
  			<img
  				src="/img/appflow.png"
  				alt="Application Flow Diagram"
  				class="diagram-preview"
  				onclick="openModal()"
  			/>
  			<p><small>Click to enlarge</small></p>
  		</div>

  		<div id="imageModal" class="modal" onclick="closeModal()">
  			<img src="/img/appflow.png" alt="Application Flow Diagram - Full Size" />
  		</div>

  		<p style="margin-top: 40px;">
  			<a href="/stats" class="link">Explore this deployment →</a>
  		</p>

  		<script>
  			function openModal() {
  				document.getElementById('imageModal').classList.add('active');
  			}
  			function closeModal() {
  				document.getElementById('imageModal').classList.remove('active');
  			}
  		</script>
  	</body>
  </html>`);
});

app.get("/stats", (_req, res) => {
	const stats = {
		hostname: os.hostname(),
		platform: os.platform(),
		arch: os.arch(),
		nodeVersion: process.version,
		uptime: Math.floor(process.uptime()),
		memory: process.memoryUsage(),
		env: {
			displayValue: process.env.DISPLAY_VALUE || "not set",
			port: process.env.PORT || "3000",
			namespace: process.env.KUBERNETES_NAMESPACE || "not set",
			podName: process.env.HOSTNAME || "not set",
			nodeName: process.env.NODE_NAME || "not set",
			podIP: process.env.POD_IP || "not set",
		}
	};

	res.set("Content-Type", "text/html");
	res.send(`<!doctype html>
<html>
  	<head>
  	  <meta charset="utf-8" />
  	  <title>Stats for Nerds - Pulumi Demo</title>
  	  <meta name="viewport" content="width=device-width, initial-scale=1" />
  	<style>
  		body { font-family: 'Monaco', 'Menlo', 'Consolas', monospace; margin: 40px; background: #1a1a1a; color: #00ff00; }
  		h1 { color: #00ff00; border-bottom: 2px solid #00ff00; padding-bottom: 10px; }
  		h2 { color: #00ccff; margin-top: 30px; border-bottom: 1px solid #00ccff; padding-bottom: 5px; }
  		.section { margin: 20px 0; background: #0d0d0d; padding: 15px; border-left: 3px solid #00ff00; }
  		.label { color: #ffff00; display: inline-block; width: 200px; }
  		.value { color: #00ff00; }
  		code { background: #0d0d0d; padding: 2px 6px; border: 1px solid #333; color: #ff6600; }
  		.link { color: #00ccff; text-decoration: none; border-bottom: 1px solid #00ccff; }
  		.link:hover { color: #00aacc; border-bottom-color: #00aacc; }
  		table { width: 100%; border-collapse: collapse; margin: 15px 0; }
  		th { background: #0d0d0d; color: #ffff00; text-align: left; padding: 10px; border: 1px solid #333; }
  		td { padding: 10px; border: 1px solid #333; color: #00ff00; }
  		tr:hover { background: #0d0d0d; }
  	</style>
  	</head>
  	<body>
  		<h1>⚙️ Stats for Nerds</h1>
  		<p><a href="/" class="link">← Back to main page</a></p>

  		<div class="section">
  			<h2>🏗️ Infrastructure</h2>
  			<table>
  				<tr><th>Component</th><th>Value</th></tr>
  				<tr><td class="label">Cloud Provider</td><td class="value">AWS (us-west-2)</td></tr>
  				<tr><td class="label">Kubernetes Platform</td><td class="value">Amazon EKS</td></tr>
  				<tr><td class="label">Cluster Size</td><td class="value">2 nodes (t3.medium)</td></tr>
  				<tr><td class="label">Container Registry</td><td class="value">Amazon ECR</td></tr>
  				<tr><td class="label">Load Balancer</td><td class="value">AWS Network Load Balancer</td></tr>
  				<tr><td class="label">DNS</td><td class="value">Route53 (pulumidemo.t8rsk8s.io)</td></tr>
  				<tr><td class="label">SSL/TLS</td><td class="value">AWS Certificate Manager (ACM)</td></tr>
  			</table>
  		</div>

  		<div class="section">
  			<h2>🐳 Container Runtime</h2>
  			<table>
  				<tr><th>Component</th><th>Value</th></tr>
  				<tr><td class="label">Base Image</td><td class="value">node:20-alpine</td></tr>
  				<tr><td class="label">Node.js Version</td><td class="value"><code>${stats.nodeVersion}</code></td></tr>
  				<tr><td class="label">Runtime User</td><td class="value">nodejs (UID 1001)</td></tr>
  				<tr><td class="label">Platform</td><td class="value">${stats.platform} / ${stats.arch}</td></tr>
  				<tr><td class="label">Container Uptime</td><td class="value">${stats.uptime} seconds</td></tr>
  			</table>
  		</div>

  		<div class="section">
  			<h2>☸️ Kubernetes Pod Info</h2>
  			<table>
  				<tr><th>Property</th><th>Value</th></tr>
  				<tr><td class="label">Pod Hostname</td><td class="value"><code>${stats.hostname}</code></td></tr>
  				<tr><td class="label">Pod Name</td><td class="value"><code>${stats.env.podName}</code></td></tr>
  				<tr><td class="label">Namespace</td><td class="value"><code>${stats.env.namespace}</code></td></tr>
  				<tr><td class="label">Node Name</td><td class="value"><code>${stats.env.nodeName}</code></td></tr>
  				<tr><td class="label">Pod IP</td><td class="value"><code>${stats.env.podIP}</code></td></tr>
  				<tr><td class="label">Replicas</td><td class="value">2 pods (managed by Deployment)</td></tr>
  			</table>
  		</div>

  		<div class="section">
  			<h2>🔧 Application Configuration</h2>
  			<table>
  				<tr><th>Variable</th><th>Value</th></tr>
  				<tr><td class="label">DISPLAY_VALUE</td><td class="value"><code>${stats.env.displayValue}</code></td></tr>
  				<tr><td class="label">PORT</td><td class="value"><code>${stats.env.port}</code></td></tr>
  			</table>
  		</div>

  		<div class="section">
  			<h2>💾 Memory Usage</h2>
  			<table>
  				<tr><th>Metric</th><th>Value</th></tr>
  				<tr><td class="label">RSS (Resident Set Size)</td><td class="value">${(stats.memory.rss / 1024 / 1024).toFixed(2)} MB</td></tr>
  				<tr><td class="label">Heap Total</td><td class="value">${(stats.memory.heapTotal / 1024 / 1024).toFixed(2)} MB</td></tr>
  				<tr><td class="label">Heap Used</td><td class="value">${(stats.memory.heapUsed / 1024 / 1024).toFixed(2)} MB</td></tr>
  				<tr><td class="label">External</td><td class="value">${(stats.memory.external / 1024 / 1024).toFixed(2)} MB</td></tr>
  			</table>
  		</div>

  		<div class="section">
  			<h2>🚀 Deployment Stack</h2>
  			<table>
  				<tr><th>Layer</th><th>Technology</th></tr>
  				<tr><td class="label">IaC Tool</td><td class="value">Pulumi (Go SDK)</td></tr>
  				<tr><td class="label">Orchestration</td><td class="value">Kubernetes 1.x (EKS)</td></tr>
  				<tr><td class="label">Web Framework</td><td class="value">Express.js</td></tr>
  				<tr><td class="label">Container Runtime</td><td class="value">containerd (via EKS)</td></tr>
  				<tr><td class="label">Networking</td><td class="value">AWS VPC CNI Plugin</td></tr>
  			</table>
  		</div>

  		<div class="section">
  			<h2>🔐 Security Posture</h2>
  			<table>
  				<tr><th>Feature</th><th>Status</th></tr>
  				<tr><td class="label">Non-root Container</td><td class="value">✅ Running as UID 1001</td></tr>
  				<tr><td class="label">TLS/SSL</td><td class="value">✅ ACM Certificate</td></tr>
  				<tr><td class="label">Private Subnets</td><td class="value">✅ Pods in private subnets</td></tr>
  				<tr><td class="label">Registry</td><td class="value">✅ Private ECR repository</td></tr>
  				<tr><td class="label">DNS Validation</td><td class="value">✅ Route53 domain ownership</td></tr>
  			</table>
  		</div>

  		<div class="section">
  			<h2>📊 Request Flow</h2>
  			<pre style="background: #0d0d0d; padding: 15px; border: 1px solid #333; overflow-x: auto;">
Browser (HTTPS:443)
    ↓
Route53 DNS (pulumidemo.t8rsk8s.io)
    ↓
Network Load Balancer (SSL Termination)
    ↓
Kubernetes Service (LoadBalancer)
    ↓
Kubernetes Service (ClusterIP:80)
    ↓
Pod (Port 3000) → Express.js → Response
  			</pre>
  		</div>

  		<p style="margin-top: 40px;">
  			<a href="/" class="link">← Back to main page</a>
  		</p>
  	</body>
  </html>`);
});


const port = process.env.PORT || 3000;
app.listen(port, () => console.log(`Listening on :${port}`));
