import React, { useState } from 'react';

const samplePrompt = `Could you please help me write a function in Go that takes a list of URLs and downloads them concurrently using goroutines? 

Please make sure to handle errors properly and kindly include comments explaining how the worker pool works. I would really appreciate it if you could also ensure that the code compiles cleanly. Thank you so much!`;

export const Playground: React.FC = () => {
  const [prompt, setPrompt] = useState('');
  const [optimizedText, setOptimizedText] = useState<string | null>(null);
  const [optimizedTokens, setOptimizedTokens] = useState<number>(0);
  const [savingsBanner, setSavingsBanner] = useState<string | null>(null);
  const [isCompressing, setIsCompressing] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [apiResponse, setApiResponse] = useState<{ status: string; data: unknown } | null>(null);

  const estimateTokens = (text: string) => {
    if (!text.trim()) return 0;
    return Math.ceil(text.trim().split(/\s+/).length * 1.33);
  };

  const handleOptimize = async () => {
    if (!prompt.trim()) return;
    try {
      setIsCompressing(true);
      const res = await fetch('/api/playground', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ prompt: prompt.trim() }),
      });
      const data = await res.json();
      setOptimizedText(data.optimized_text);
      setOptimizedTokens(data.optimized_tokens);
      setSavingsBanner(
        `Optimized: ${data.saved_tokens} tokens saved (${data.savings_percent.toFixed(1)}% reduction)`
      );
    } catch (err: unknown) {
      setOptimizedText(`Optimization error: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setIsCompressing(false);
    }
  };

  const handleExecute = async () => {
    if (!prompt.trim()) {
      alert('Please enter a prompt first.');
      return;
    }

    try {
      setIsSending(true);
      const res = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: 'gpt-4o',
          messages: [{ role: 'user', content: prompt.trim() }],
        }),
      });

      const isCacheHit = res.headers.get('X-Cache') === 'HIT';
      const data = await res.json();
      setApiResponse({
        status: `${res.status} ${res.statusText} • Cache: ${isCacheHit ? 'HIT (0 tokens billed)' : 'MISS'}`,
        data,
      });
    } catch (err: unknown) {
      setApiResponse({
        status: 'Error',
        data: { error: err instanceof Error ? err.message : String(err) },
      });
    } finally {
      setIsSending(false);
    }
  };

  return (
    <div className="dashboard-card">
      <div className="card-title-row">
        <div>
          <h2 className="card-heading">🧪 Interactive Token Optimizer Playground</h2>
          <p className="card-subheading">
            Evaluate real-time token compression, prompt efficiency, and semantic response caching.
          </p>
        </div>
        <button
          className="btn-secondary"
          onClick={() => {
            setPrompt(samplePrompt);
            setOptimizedText(null);
            setSavingsBanner(null);
          }}
        >
          Load Sample Prompt
        </button>
      </div>

      <div className="playground-layout">
        <div className="playground-pane">
          <div className="pane-header">
            <span>Original Prompt (Unoptimized)</span>
            <span className="token-badge badge-orig">
              Tokens: {estimateTokens(prompt)}
            </span>
          </div>
          <textarea
            className="playground-input"
            placeholder="Type or paste any prompt to test token optimization..."
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
          />
          <div className="pane-actions">
            <button
              className="btn-primary"
              disabled={isCompressing || !prompt.trim()}
              onClick={handleOptimize}
            >
              {isCompressing ? '⚡ Optimizing...' : '⚡ Optimize Prompt'}
            </button>
            <button
              className="btn-secondary"
              disabled={isSending || !prompt.trim()}
              onClick={handleExecute}
            >
              {isSending ? '⏳ Running...' : '🚀 Test Execution'}
            </button>
          </div>
        </div>

        <div className="playground-pane">
          <div className="pane-header">
            <span>Optimized Output</span>
            <div>
              <span className="token-badge badge-opt">
                Tokens: {optimizedTokens}
              </span>
            </div>
          </div>
          <div className="output-preview">
            {optimizedText ? (
              optimizedText
            ) : (
              <span className="placeholder-text">
                Click "Optimize Prompt" or test execution to preview the optimized payload...
              </span>
            )}
          </div>
          {savingsBanner && (
            <div className="savings-banner">
              <span>⚡</span>
              <span>{savingsBanner}</span>
            </div>
          )}
        </div>
      </div>

      {apiResponse && (
        <div className="api-response-box">
          <div className="pane-header">
            <span>Live API Response</span>
            <span className="badge-tag" style={{ color: '#38bdf8', borderColor: 'rgba(56,189,248,0.4)' }}>
              {apiResponse.status}
            </span>
          </div>
          <pre className="response-code">
            <code>{JSON.stringify(apiResponse.data, null, 2)}</code>
          </pre>
        </div>
      )}
    </div>
  );
};
