FROM node:20-alpine

WORKDIR /app

# Create the webhook receiver server
RUN cat > server.js << 'EOF'
const http = require("http");
const fs = require("fs");
const path = require("path");

const logDir = "/app/logs";
if (!fs.existsSync(logDir)) {
  fs.mkdirSync(logDir, { recursive: true });
}

const server = http.createServer((req, res) => {
  let body = "";
  req.on("data", chunk => { body += chunk; });
  req.on("end", () => {
    const timestamp = new Date().toISOString();
    const logFile = path.join(logDir, `webhooks-${new Date().toISOString().split("T")[0]}.log`);
    const logEntry = `[${timestamp}] ${req.method} ${req.url}\nHeaders: ${JSON.stringify(req.headers)}\nBody: ${body}\n\n`;
    
    fs.appendFileSync(logFile, logEntry);
    console.log(`[${timestamp}] Webhook received: ${req.method} ${req.url}`);
    
    res.writeHead(200, {"Content-Type": "application/json"});
    res.end(JSON.stringify({ status: "ok", received: timestamp }));
  });
});

server.listen(8080, "0.0.0.0", () => {
  console.log("Webhook receiver listening on port 8080");
});
EOF

EXPOSE 8080

CMD ["node", "server.js"]
