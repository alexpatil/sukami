import React, { useState } from 'react';

export const Docs: React.FC = () => {
  const [lang, setLang] = useState<'python' | 'node' | 'curl'>('python');

  return (
    <div className="dashboard-card">
      <div className="card-title-row">
        <div>
          <h2 className="card-heading">🔌 1-Line Drop-in Integration</h2>
          <p className="card-subheading">
            Zero migration effort. Redirect your client's <code>base_url</code> to Sukami to instantly save 30% to 100% on tokens.
          </p>
        </div>
      </div>

      <div style={{ marginBottom: 20, padding: '12px 16px', background: 'rgba(56, 189, 248, 0.08)', border: '1px solid rgba(56, 189, 248, 0.2)', borderRadius: 8, fontSize: '0.85rem', color: '#94a3b8' }}>
        <strong style={{ color: '#38bdf8' }}>🎁 Free Trial Active:</strong> Every new API key includes <strong>100,000 free tokens</strong> with automatic prompt pruning and output fluff stripping enabled.
      </div>

      <div className="code-tabs">
        <button
          className={`code-tab ${lang === 'python' ? 'active' : ''}`}
          onClick={() => setLang('python')}
        >
          Python
        </button>
        <button
          className={`code-tab ${lang === 'node' ? 'active' : ''}`}
          onClick={() => setLang('node')}
        >
          Node.js
        </button>
        <button
          className={`code-tab ${lang === 'curl' ? 'active' : ''}`}
          onClick={() => setLang('curl')}
        >
          cURL
        </button>
      </div>

      {lang === 'python' && (
        <div className="code-content">
          <pre className="code-box">
            <code>{`from openai import OpenAI

client = OpenAI(
    base_url="https://api.sukami.io/v1",  # Or http://localhost:8080/v1 for local testing
    api_key="sk-sukami-your-secret-key"   # Generated from API Keys tab (100k free trial tokens)
)

response = client.chat.completions.create(
    model="gpt-4o",
    messages=[{"role": "user", "content": "Explain distributed systems architecture"}]
)

# Returns dense, high-performance response with up to 50% lower output costs
print(response.choices[0].message.content)`}</code>
          </pre>
        </div>
      )}

      {lang === 'node' && (
        <div className="code-content">
          <pre className="code-box">
            <code>{`import OpenAI from 'openai';

const openai = new OpenAI({
  baseURL: 'https://api.sukami.io/v1', // Or http://localhost:8080/v1 for local testing
  apiKey: process.env.SUKAMI_API_KEY,   // sk-sukami-...
});

const response = await openai.chat.completions.create({
  model: 'gpt-4o',
  messages: [{ role: 'user', content: 'Explain distributed systems architecture' }],
});

console.log(response.choices[0].message.content);`}</code>
          </pre>
        </div>
      )}

      {lang === 'curl' && (
        <div className="code-content">
          <pre className="code-box">
            <code>{`curl https://api.sukami.io/v1/chat/completions \\
  -H "Content-Type: application/json" \\
  -H "Authorization: Bearer sk-sukami-your-secret-key" \\
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "Explain distributed systems architecture"}]
  }'`}</code>
          </pre>
        </div>
      )}
    </div>
  );
};
