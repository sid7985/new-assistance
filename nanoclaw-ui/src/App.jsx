import { useState, useEffect } from 'react'

function App() {
  const [budget, setBudget] = useState({ limitCents: 1000, spentCents: 0 })
  const [missions, setMissions] = useState([])
  const [todos, setTodos] = useState([])
  const [auditLogs, setAuditLogs] = useState([])

  useEffect(() => {
    // Poll the Nanoclaw Orchestrator API every 2 seconds
    const fetchData = async () => {
      try {
        const budgetRes = await fetch('http://localhost:8080/api/budget')
        if (budgetRes.ok) setBudget(await budgetRes.json())

        const missionsRes = await fetch('http://localhost:8080/api/missions')
        if (missionsRes.ok) setMissions(await missionsRes.json())

        const todosRes = await fetch('http://localhost:8080/api/todos')
        if (todosRes.ok) setTodos(await todosRes.json())

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
      <header className="flex justify-between items-center mb-8 bg-surface backdrop-blur-md p-4 rounded-xl border border-white/10 shadow-2xl">
        <div className="flex items-center gap-4">
          <div className="w-10 h-10 bg-primary rounded-lg flex items-center justify-center font-bold text-xl shadow-lg border border-white/20">Ω</div>
          <h1 className="text-2xl font-bold bg-gradient-to-r from-primary to-blue-400 bg-clip-text text-transparent">
            OmniClaw Agency
          </h1>
        </div>
        <div className="flex items-center gap-6">
          <div className="flex flex-col items-end">
            <span className="text-xs uppercase tracking-widest text-gray-400 font-semibold mb-1">Agency Spend</span>
            <span className="font-mono text-lg">${(budget.spentCents / 100).toFixed(2)} <span className="text-gray-500 text-sm">/ ${(budget.limitCents / 100).toFixed(2)}</span></span>
          </div>
          <div className="w-40 h-3 bg-gray-800 rounded-full overflow-hidden border border-white/5 p-[1px]">
            <div 
              className="h-full bg-gradient-to-r from-primary to-blue-500 rounded-full shadow-[0_0_10px_rgba(59,130,246,0.5)] transition-all duration-1000" 
              style={{ width: `${Math.min(100, (budget.spentCents / budget.limitCents) * 100)}%` }}
            ></div>
          </div>
        </div>
      </header>

      <div className="flex flex-1 gap-6 min-h-0">
        {/* Sidebar: Missions & Agency Structure */}
        <aside className="w-[400px] flex flex-col gap-4 overflow-hidden">
          <div className="flex justify-between items-end mb-2">
            <h2 className="text-xl font-semibold">Active Missions</h2>
            <span className="text-xs text-primary font-mono">{missions.length} active</span>
          </div>
          <div className="flex-1 overflow-y-auto space-y-4 pr-2 custom-scrollbar">
            {missions.length === 0 ? (
              <div className="p-10 text-center border-2 border-dashed border-white/5 rounded-xl">
                <p className="text-gray-500 italic">Agency idle...</p>
              </div>
            ) : (
              missions.map(m => (
                <div key={m.id} className="p-5 bg-surface rounded-xl border border-white/5 shadow-xl transition-all hover:border-primary/30 group">
                  <div className="flex justify-between items-start mb-3">
                    <div className="flex items-center gap-2">
                      <span className="text-[10px] font-mono bg-white/5 px-2 py-1 rounded text-gray-500">#{m.id}</span>
                      <span className={`text-[10px] px-2 py-0.5 rounded uppercase font-bold tracking-tighter ${m.status === 'active' ? 'bg-blue-500/20 text-blue-400 border border-blue-500/30' : m.status === 'completed' ? 'bg-green-500/20 text-green-400 border border-green-500/30' : 'bg-red-500/20 text-red-400 border border-red-500/30'}`}>
                        {m.status}
                      </span>
                    </div>
                    <span className="text-[10px] text-gray-600 font-mono">{new Date(m.created_at).toLocaleTimeString()}</span>
                  </div>
                  <h3 className="text-sm font-medium mb-4 group-hover:text-primary transition-colors">{m.goal}</h3>
                  
                  {/* Granular TODOs (Agency/Claw style) */}
                  <div className="space-y-2 mb-4 border-l-2 border-white/5 pl-4 ml-1">
                    {todos.filter(t => t.mission_id === m.id).map(todo => (
                      <div key={todo.id} className="flex items-start gap-3 text-xs">
                        <span className="mt-0.5">
                          {todo.status === 'completed' ? '✅' : todo.status === 'running' ? '🏃' : todo.status === 'failed' ? '❌' : '⏳'}
                        </span>
                        <div className="flex-1">
                          <span className="text-[10px] font-mono text-primary/70 uppercase">[{todo.worker_type}]</span>{" "}
                          <span className={todo.status === 'completed' ? 'text-gray-400 line-through' : 'text-gray-200'}>
                            {todo.task_goal}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>

                  <div className="pt-3 border-t border-white/5 flex justify-between text-[10px] text-gray-500 font-mono">
                    <div className="flex gap-4">
                      <span className="flex items-center gap-1"><div className="w-1.5 h-1.5 rounded-full bg-blue-500"></div> {m.api_calls} calls</span>
                      <span className="flex items-center gap-1"><div className="w-1.5 h-1.5 rounded-full bg-primary"></div> {m.tokens_used} tkns</span>
                    </div>
                  </div>
                </div>
              ))
            )}
          </div>
        </aside>

        {/* Main: Audit Log Terminal */}
        <main className="flex-1 bg-[#0a0a0f] rounded-2xl border border-white/10 flex flex-col overflow-hidden shadow-2xl">
          <div className="bg-white/5 py-3 px-6 border-b border-white/10 flex justify-between items-center">
            <div className="flex gap-2">
              <div className="w-3 h-3 rounded-full bg-red-500 shadow-[0_0_5px_rgba(239,68,68,0.5)]"></div>
              <div className="w-3 h-3 rounded-full bg-yellow-500 shadow-[0_0_5px_rgba(245,158,11,0.5)]"></div>
              <div className="w-3 h-3 rounded-full bg-green-500 shadow-[0_0_5px_rgba(34,197,94,0.5)]"></div>
              <span className="ml-4 text-xs text-gray-400 font-mono tracking-widest uppercase">agency_intelligence.log</span>
            </div>
          </div>
          
          <div className="flex-1 p-6 font-mono text-xs overflow-y-auto custom-scrollbar">
            {auditLogs.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-full text-gray-700 opacity-20 transform -rotate-12">
                <span className="text-6xl font-bold">NANOCLAW</span>
                <span className="text-xl">Standby for instructions...</span>
              </div>
            ) : (
              auditLogs.map((log, index) => (
                <div key={log.id} className={`mb-3 p-2 rounded transition-colors ${index === 0 ? 'bg-primary/5 border-l-2 border-primary' : 'hover:bg-white/5'}`}>
                  <div className="flex items-center gap-3 mb-1">
                    <span className="text-gray-600 text-[10px]">[{new Date(log.created_at).toLocaleTimeString()}]</span>
                    <span className="text-primary font-bold tracking-tighter uppercase px-1.5 py-0.5 bg-primary/10 rounded-sm">{log.source}</span>
                    <span className={`text-[10px] font-bold px-1.5 py-0.5 rounded-sm ${log.action_type === 'ERROR' ? 'bg-red-500/20 text-red-500' : 'bg-green-500/20 text-green-500'}`}>
                      {log.action_type}
                    </span>
                    {log.tokens_used > 0 && <span className="text-gray-600 ml-auto">{log.tokens_used} tokens</span>}
                  </div>
                  <div className="text-gray-300 leading-relaxed pl-2 border-l border-white/5">
                    {log.action_detail}
                  </div>
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
