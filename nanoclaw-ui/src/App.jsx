import { useState, useEffect } from 'react'

function App() {
  const [budget, setBudget] = useState({ limitCents: 1000, spentCents: 0 })
  const [missions, setMissions] = useState([])
  const [auditLogs, setAuditLogs] = useState([])

  useEffect(() => {
    // Poll the Nanoclaw Orchestrator API every 2 seconds
    const fetchData = async () => {
      try {
        const budgetRes = await fetch('http://localhost:8080/api/budget')
        if (budgetRes.ok) setBudget(await budgetRes.json())

        const missionsRes = await fetch('http://localhost:8080/api/missions')
        if (missionsRes.ok) setMissions(await missionsRes.json())

        const auditRes = await fetch('http://localhost:8080/api/audit')
        if (auditRes.ok) setAuditLogs(await auditRes.json())
      } catch (err) {
        console.error("API not reachable yet", err)
      }
    }

    fetchData()
    const interval = setInterval(fetchData, 2000)
    return () => clearInterval(interval)
  }, [])

  return (
    <div className="min-h-screen bg-bg text-white w-full flex flex-col p-6">
      {/* Header */}
      <header className="flex justify-between items-center mb-8 bg-surface backdrop-blur-md p-4 rounded-xl border border-white/10">
        <h1 className="text-2xl font-bold bg-gradient-to-r from-primary to-blue-400 bg-clip-text text-transparent">
          OmniClaw Dashboard
        </h1>
        <div className="flex items-center gap-4">
          <div className="flex flex-col items-end">
            <span className="text-sm text-gray-400">Monthly Budget</span>
            <span className="font-mono">${(budget.spentCents / 100).toFixed(2)} / ${(budget.limitCents / 100).toFixed(2)}</span>
          </div>
          <div className="w-32 h-2 bg-gray-800 rounded-full overflow-hidden">
            <div 
              className="h-full bg-primary" 
              style={{ width: `${Math.min(100, (budget.spentCents / budget.limitCents) * 100)}%` }}
            ></div>
          </div>
        </div>
      </header>

      <div className="flex flex-1 gap-6">
        {/* Sidebar: Missions */}
        <aside className="w-1/3 flex flex-col gap-4">
          <h2 className="text-xl font-semibold mb-2">Active Missions</h2>
          <div className="flex-1 overflow-y-auto space-y-3">
            {missions.length === 0 ? (
              <p className="text-gray-500 italic">No missions running...</p>
            ) : (
              missions.map(m => (
                <div key={m.id} className="p-4 bg-surface rounded-lg border border-white/5 shadow-lg">
                  <div className="flex justify-between items-start mb-2">
                    <span className="text-xs font-mono text-gray-400">#{m.id}</span>
                    <span className={`text-xs px-2 py-1 rounded-full ${m.status === 'active' ? 'bg-blue-500/20 text-blue-300' : m.status === 'completed' ? 'bg-green-500/20 text-green-300' : 'bg-red-500/20 text-red-300'}`}>
                      {m.status}
                    </span>
                  </div>
                  <p className="text-sm">{m.goal}</p>
                  <div className="mt-3 flex justify-between text-xs text-gray-500 font-mono">
                    <span>{m.api_calls} calls</span>
                    <span>{m.tokens_used} tkns</span>
                  </div>
                </div>
              ))
            )}
          </div>
        </aside>

        {/* Main: Audit Log Terminal */}
        <main className="flex-1 bg-[#0a0a0f] rounded-xl border border-white/10 flex flex-col overflow-hidden shadow-2xl relative">
          <div className="bg-white/5 py-2 px-4 border-b border-white/10 flex gap-2">
            <div className="w-3 h-3 rounded-full bg-red-500"></div>
            <div className="w-3 h-3 rounded-full bg-yellow-500"></div>
            <div className="w-3 h-3 rounded-full bg-green-500"></div>
            <span className="ml-4 text-xs text-gray-400 font-mono">system.log</span>
          </div>
          
          <div className="flex-1 p-4 font-mono text-sm overflow-y-auto">
            {auditLogs.length === 0 ? (
              <span className="text-gray-600">Waiting for agent activity...</span>
            ) : (
              auditLogs.map(log => (
                <div key={log.id} className="mb-2 hover:bg-white/5 p-1 rounded">
                  <span className="text-gray-500">[{new Date(log.created_at).toLocaleTimeString()}]</span>{" "}
                  <span className="text-blue-400">{log.source.toUpperCase()}</span>{" "}
                  <span className={log.action_type === 'ERROR' ? 'text-red-400' : log.action_type === 'COMPLETED' ? 'text-green-400' : 'text-gray-300'}>
                    [{log.action_type}]
                  </span>{" "}
                  <span className="text-gray-200">{log.action_detail}</span>
                  {log.tokens_used > 0 && <span className="text-gray-600 ml-2">({log.tokens_used}t)</span>}
                </div>
              ))
            )}
          </div>
        </main>
      </div>
    </div>
  )
}

export default App
