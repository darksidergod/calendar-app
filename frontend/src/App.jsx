import { useState, useEffect, useCallback } from 'react'

async function handleAuthRequired(res) {
  if (res.status === 403) {
    const authRes = await fetch('/authurl')
    if (!authRes.ok) throw new Error('Failed to get auth URL')
    const { url } = await authRes.json()
    if (url) window.location.href = url
    return true
  }
  return false
}

function App() {
  const [events, setEvents] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const fetchEvents = useCallback(async () => {
    try {
      const res = await fetch('/listevents')
      if (await handleAuthRequired(res)) return
      const text = await res.text()
      let data = {}
      try {
        data = JSON.parse(text)
      } catch {}
      if (!res.ok) {
        throw new Error(data?.error || data?.message || text || res.statusText)
      }
      setEvents(data?.events ?? [])
      setError(null)
    } catch (err) {
      setError(err.message)
      setEvents([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchEvents()
  }, [fetchEvents])

  const checkAuthAndRetry = async (res, onSuccess) => {
    if (await handleAuthRequired(res)) return
    if (!res.ok) {
      const text = await res.text()
      let msg = res.statusText
      try {
        const data = JSON.parse(text)
        msg = data?.error || data?.message || msg
      } catch {}
      throw new Error(msg)
    }
    onSuccess?.()
  }

  const handleDelete = async (eventId) => {
    try {
      const res = await fetch(`/deleteevent?eventId=${encodeURIComponent(eventId)}`, {
        method: 'POST',
      })
      await checkAuthAndRetry(res, fetchEvents)
    } catch (err) {
      setError(err.message)
    }
  }

  const handleDeleteAll = async (eventIds) => {
    try {
      for (const id of eventIds) {
        const res = await fetch(`/deleteevent?eventId=${encodeURIComponent(id)}`, { method: 'POST' })
        if (await handleAuthRequired(res)) return
        if (!res.ok) {
          const text = await res.text()
          let msg = res.statusText
          try {
            const data = JSON.parse(text)
            msg = data?.error || data?.message || msg
          } catch {}
          throw new Error(msg)
        }
      }
      fetchEvents()
    } catch (err) {
      setError(err.message)
    }
  }

  const handleCreate = async (eventsToCreate) => {
    try {
      const res = await fetch('/createevents', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(eventsToCreate),
      })
      await checkAuthAndRetry(res, fetchEvents)
    } catch (err) {
      setError(err.message)
    }
  }

  const refreshToken = async () => {
    try {
      const res = await fetch('/authurl')
      if (!res.ok) throw new Error('Failed to get auth URL')
      const { url } = await res.json()
      if (url) window.location.href = url
    } catch (err) {
      setError(err.message)
    }
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1rem' }}>
        <h1 style={{ margin: 0 }}>Calendar</h1>
        <button
          type="button"
          onClick={refreshToken}
          style={{
            padding: '0.35rem 0.75rem',
            fontSize: '0.9em',
            border: '1px solid #666',
            background: 'white',
            borderRadius: '4px',
            cursor: 'pointer',
          }}
        >
          Refresh token
        </button>
      </div>
      <CreateEventForm onSubmit={handleCreate} />
      {loading && <p>Loading events…</p>}
      {error && (
        <div
          role="alert"
          style={{
            padding: '0.75rem 1rem',
            marginBottom: '1rem',
            background: '#fee',
            border: '1px solid #c00',
            borderRadius: '6px',
            color: '#c00',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            gap: '1rem',
          }}
        >
          <span>Error: {error}</span>
          <button
            type="button"
            onClick={() => setError(null)}
            style={{
              padding: '0.25rem 0.5rem',
              border: '1px solid #c00',
              background: 'white',
              color: '#c00',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '0.9em',
            }}
          >
            Dismiss
          </button>
        </div>
      )}
      {!loading && !error && events.length === 0 && <p>No events.</p>}
      {!loading && !error && events.length > 0 && (
        <EventGroups events={events} onDelete={handleDelete} onDeleteAll={handleDeleteAll} />
      )}
    </div>
  )
}

// IANA time zones; ensure Asia/Kolkata and common zones are included
const REQUIRED_ZONES = ['Asia/Kolkata', 'UTC']
const TIME_ZONES = (() => {
  let list = []
  try {
    if (typeof Intl?.supportedValuesOf === 'function') {
      list = Intl.supportedValuesOf('timeZone')
    }
  } catch (_) {}
  if (list.length === 0) {
    list = [
      'Africa/Cairo', 'America/Chicago', 'America/New_York', 'Asia/Dubai', 'Asia/Hong_Kong',
      'Asia/Kolkata', 'Asia/Seoul', 'Asia/Tokyo', 'Europe/London', 'Europe/Paris', 'UTC',
    ]
  }
  return [...new Set([...list, ...REQUIRED_ZONES])].sort()
})()

function toRFC3339(datetime) {
  if (!datetime) return ''
  return datetime.length === 16 ? `${datetime}:00` : datetime
}

const INITIAL_EVENT = {
  summary: '',
  description: '',
  allDay: false,
  dayOffset: 0,
  startTime: '09:00',
  endTime: '10:00',
  timeZone: 'UTC',
}

function addDays(dateStr, days) {
  const d = new Date(dateStr + 'T12:00:00')
  d.setDate(d.getDate() + days)
  return d.toISOString().slice(0, 10)
}

const TEMPLATE_KEY = 'calendar-app-templates'

function CreateEventForm({ onSubmit }) {
  const [createError, setCreateError] = useState(null)
  const [topic, setTopic] = useState('')
  const [mainDate, setMainDate] = useState(() => new Date().toISOString().slice(0, 10))
  const [sameTimeZone, setSameTimeZone] = useState(false)
  const [sharedTimeZone, setSharedTimeZone] = useState('UTC')
  const [events, setEvents] = useState([{ ...INITIAL_EVENT }])
  const [templates, setTemplates] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem(TEMPLATE_KEY) || '[]')
    } catch {
      return []
    }
  })
  const [selectedTemplate, setSelectedTemplate] = useState('')

  const addEvent = () => setEvents((e) => [...e, { ...INITIAL_EVENT }])
  const addEventAfter = (idx) => setEvents((e) => {
    const next = [...e]
    next.splice(idx + 1, 0, { ...INITIAL_EVENT })
    return next
  })
  const removeEvent = (idx) => setEvents((e) => (e.length > 1 ? e.filter((_, i) => i !== idx) : e))
  const updateEvent = (idx, patch) =>
    setEvents((e) => e.map((ev, i) => (i === idx ? { ...ev, ...patch } : ev)))

  const saveTemplate = () => {
    const name = prompt('Template name:')
    if (!name?.trim()) return
    const t = {
      name: name.trim(),
      topic,
      sameTimeZone,
      sharedTimeZone,
      events: events.map(({ summary, description, allDay, dayOffset, startTime, endTime, timeZone }) =>
        ({ summary, description, allDay, dayOffset, startTime, endTime, timeZone })),
    }
    const next = [...templates.filter((x) => x.name !== t.name), t]
    setTemplates(next)
    localStorage.setItem(TEMPLATE_KEY, JSON.stringify(next))
    setSelectedTemplate(t.name)
  }

  const loadTemplate = (name) => {
    if (!name) return
    const t = templates.find((x) => x.name === name)
    if (!t) return
    setTopic(t.topic || '')
    setSameTimeZone(t.sameTimeZone ?? false)
    setSharedTimeZone(t.sharedTimeZone || 'UTC')
    setEvents(t.events.map((e) => ({ ...INITIAL_EVENT, ...e })))
    setSelectedTemplate(name)
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    setCreateError(null)
    if (!topic.trim()) {
      setCreateError('Topic is required')
      return
    }
    if (!mainDate) {
      setCreateError('Main date is required')
      return
    }
    const prefix = topic.trim()
    const built = []
    for (let i = 0; i < events.length; i++) {
      const ev = events[i]
      if (!ev.summary.trim()) {
        setCreateError(`Event ${i + 1}: Summary is required`)
        return
      }
      if (!ev.description.trim()) {
        setCreateError(`Event ${i + 1}: Description is required`)
        return
      }
      const dateStr = addDays(mainDate, ev.dayOffset ?? 0)
      const base = {
        summary: `${prefix}: ${ev.summary.trim()}`,
        description: ev.description.trim(),
      }
      if (ev.allDay) {
        built.push({ ...base, start: { date: dateStr }, end: { date: dateStr } })
      } else {
        const startDt = `${dateStr}T${(ev.startTime || '09:00').slice(0, 5)}:00`
        const endDt = `${dateStr}T${(ev.endTime || '10:00').slice(0, 5)}:00`
        const tz = sameTimeZone ? sharedTimeZone : (ev.timeZone || 'UTC')
        built.push({
          ...base,
          start: { dateTime: toRFC3339(startDt), timeZone: tz },
          end: { dateTime: toRFC3339(endDt), timeZone: tz },
        })
      }
    }
    try {
      await onSubmit(built)
      setTopic('')
      setMainDate(new Date().toISOString().slice(0, 10))
      setEvents([{ ...INITIAL_EVENT }])
    } catch (err) {
      setCreateError(err.message)
    }
  }

  return (
    <section style={{ marginBottom: '1.5rem', padding: '1rem', background: 'white', borderRadius: '8px', boxShadow: '0 1px 3px rgba(0,0,0,0.1)' }}>
      <h2>Create events</h2>
      <form onSubmit={handleSubmit}>
        <div style={{ marginBottom: '1rem' }}>
          <label htmlFor="topic" style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>Topic (prefix for all events)</label>
          <input
            id="topic"
            type="text"
            value={topic}
            onChange={(e) => setTopic(e.target.value)}
            placeholder="e.g. Workshop, Seminar"
            required
            style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
          />
        </div>
        <div style={{ marginBottom: '1rem' }}>
          <label htmlFor="mainDate" style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>Main date</label>
          <input
            id="mainDate"
            type="date"
            value={mainDate}
            onChange={(e) => setMainDate(e.target.value)}
            required
            style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
          />
        </div>
        <div style={{ marginBottom: '1rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <input
            id="sameTimeZone"
            type="checkbox"
            checked={sameTimeZone}
            onChange={(e) => setSameTimeZone(e.target.checked)}
          />
          <label htmlFor="sameTimeZone" style={{ fontSize: '0.9em' }}>All events same time zone</label>
        </div>
        {sameTimeZone && (
          <div style={{ marginBottom: '1rem' }}>
            <label htmlFor="sharedTimeZone" style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>Time zone</label>
            <select
              id="sharedTimeZone"
              value={sharedTimeZone}
              onChange={(e) => setSharedTimeZone(e.target.value)}
              style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
            >
              {TIME_ZONES.map((tz) => (
                <option key={tz} value={tz}>{tz}</option>
              ))}
            </select>
          </div>
        )}
        <div style={{ marginBottom: '1rem', display: 'flex', gap: '0.5rem', flexWrap: 'wrap', alignItems: 'center' }}>
          <label style={{ fontSize: '0.9em' }}>Template:</label>
          <select
            value={selectedTemplate}
            onChange={(e) => loadTemplate(e.target.value)}
            style={{ padding: '0.35rem 0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
          >
            <option value="">— Load —</option>
            {templates.map((t) => (
              <option key={t.name} value={t.name}>{t.name}</option>
            ))}
          </select>
          <button type="button" onClick={saveTemplate} style={{ padding: '0.35rem 0.75rem', borderRadius: '4px', border: '1px solid #666', background: 'white', cursor: 'pointer', fontSize: '0.9em' }}>
            Save as template
          </button>
        </div>
        <hr style={{ margin: '1rem 0', border: 'none', borderTop: '1px solid #eee' }} />
        {events.map((ev, idx) => (
          <EventFormRow
            key={idx}
            index={idx}
            event={ev}
            mainDate={mainDate}
            sameTimeZone={sameTimeZone}
            sharedTimeZone={sharedTimeZone}
            onChange={(patch) => updateEvent(idx, patch)}
            onRemove={() => removeEvent(idx)}
            onAddAfter={() => addEventAfter(idx)}
            canRemove={events.length > 1}
            timeZones={TIME_ZONES}
          />
        ))}
        <div style={{ marginTop: '1rem', display: 'flex', gap: '0.5rem', flexWrap: 'wrap' }}>
          <button
            type="button"
            onClick={addEvent}
            style={{ padding: '0.5rem 1rem', borderRadius: '4px', border: '1px dashed #1976d2', background: 'white', color: '#1976d2', cursor: 'pointer' }}
          >
            + Add another event
          </button>
          <button type="submit" style={{ padding: '0.5rem 1rem', borderRadius: '4px', border: 'none', background: '#1976d2', color: 'white', cursor: 'pointer' }}>
            Create {events.length} event{events.length !== 1 ? 's' : ''}
          </button>
        </div>
        {createError && <p style={{ color: 'crimson', marginTop: '0.5rem' }}>{createError}</p>}
      </form>
    </section>
  )
}

function EventFormRow({ index, event: ev, mainDate, sameTimeZone, sharedTimeZone, onChange, onRemove, onAddAfter, canRemove, timeZones }) {
  const offset = ev.dayOffset ?? 0
  const computedDate = mainDate ? addDays(mainDate, offset) : ''
  return (
    <div style={{ marginBottom: '1.5rem', padding: '1rem', background: '#fafafa', borderRadius: '8px', border: '1px solid #eee' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '0.75rem', flexWrap: 'wrap', gap: '0.5rem' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
          <strong>Event {index + 1}</strong>
          <button type="button" onClick={onAddAfter} style={{ fontSize: '0.8em', padding: '0.2rem 0.4rem', border: '1px dashed #1976d2', background: 'white', color: '#1976d2', borderRadius: '4px', cursor: 'pointer' }} title="Add sub-event after this">
            + Add
          </button>
        </div>
        {canRemove && (
          <button type="button" onClick={onRemove} style={{ fontSize: '0.85em', padding: '0.25rem 0.5rem', border: '1px solid #ccc', background: 'white', borderRadius: '4px', cursor: 'pointer' }}>
            Remove
          </button>
        )}
      </div>
      <div style={{ marginBottom: '0.75rem' }}>
        <label style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>Summary</label>
        <input
          type="text"
          value={ev.summary}
          onChange={(e) => onChange({ summary: e.target.value })}
          placeholder="Short title"
          required
          style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
        />
      </div>
      <div style={{ marginBottom: '0.75rem' }}>
        <label style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>Description</label>
        <input
          type="text"
          value={ev.description}
          onChange={(e) => onChange({ description: e.target.value })}
          required
          style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
        />
      </div>
      <div style={{ marginBottom: '0.75rem' }}>
        <label style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>Day offset (from main date): {offset >= 0 ? '+' : ''}{offset} days → {computedDate || '—'}</label>
        <input
          type="range"
          min={-14}
          max={14}
          value={offset}
          onChange={(e) => onChange({ dayOffset: Number(e.target.value) })}
          style={{ width: '100%', accentColor: '#1976d2' }}
        />
      </div>
      <div style={{ marginBottom: '0.75rem', display: 'flex', alignItems: 'center', gap: '0.5rem' }}>
        <input
          type="checkbox"
          id={`allDay-${index}`}
          checked={ev.allDay}
          onChange={(e) => onChange({ allDay: e.target.checked })}
        />
        <label htmlFor={`allDay-${index}`} style={{ fontSize: '0.9em' }}>All-day event</label>
      </div>
      {!ev.allDay && (
        <div style={{ display: 'grid', gridTemplateColumns: sameTimeZone ? '1fr 1fr' : '1fr 1fr 1fr', gap: '0.75rem' }}>
          <div>
            <label style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>Start time</label>
            <input
              type="time"
              value={ev.startTime || '09:00'}
              onChange={(e) => onChange({ startTime: e.target.value })}
              style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
            />
          </div>
          <div>
            <label style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>End time</label>
            <input
              type="time"
              value={ev.endTime || '10:00'}
              onChange={(e) => onChange({ endTime: e.target.value })}
              style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
            />
          </div>
          {!sameTimeZone && (
            <div>
              <label style={{ display: 'block', marginBottom: '0.25rem', fontSize: '0.9em' }}>Time zone</label>
              <select
                value={ev.timeZone || 'UTC'}
                onChange={(e) => onChange({ timeZone: e.target.value })}
                style={{ width: '100%', padding: '0.5rem', borderRadius: '4px', border: '1px solid #ccc' }}
              >
                {timeZones.map((tz) => (
                  <option key={tz} value={tz}>{tz}</option>
                ))}
              </select>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function getGroupId(event) {
  const ep = event.extendedProperties?.private
  const id = ep?.['calendar-app-group-id'] ?? ep?.['groupId'] ?? event.groupId
  if (id) return id
  const s = event.summary || ''
  const colon = s.indexOf(': ')
  return colon > 0 ? s.slice(0, colon) : s || '__ungrouped'
}

function getGroupDisplayName(groupId, events) {
  if (groupId === '__ungrouped') return null
  const s = events[0]?.summary || ''
  const colon = s.indexOf(': ')
  if (colon > 0) return s.slice(0, colon)
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(groupId) ? null : groupId
}

function stripGroupIdFromDescription(description) {
  if (!description || typeof description !== 'string') return description
  return description.replace(/\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b/gi, '').replace(/\n{2,}/g, '\n').trim()
}

function EventGroups({ events, onDelete, onDeleteAll }) {
  const groups = events.reduce((acc, ev) => {
    const gid = getGroupId(ev)
    if (!acc[gid]) acc[gid] = []
    acc[gid].push(ev)
    return acc
  }, {})
  const entries = Object.entries(groups)
  return (
    <ul style={{ listStyle: 'none', padding: 0 }}>
      {entries.map(([groupId, evs]) => (
        <li key={groupId} style={{ marginBottom: '1rem' }}>
          <EventGroup groupId={groupId} events={evs} onDelete={onDelete} onDeleteAll={onDeleteAll} />
        </li>
      ))}
    </ul>
  )
}

function EventGroup({ groupId, events, onDelete, onDeleteAll }) {
  const topic = getGroupDisplayName(groupId, events)
  const [deletingAll, setDeletingAll] = useState(false)
  const handleDeleteAll = async () => {
    if (deletingAll) return
    setDeletingAll(true)
    try {
      await onDeleteAll(events.map((e) => e.id))
    } finally {
      setDeletingAll(false)
    }
  }
  return (
    <div style={{ background: 'white', borderRadius: '8px', boxShadow: '0 1px 3px rgba(0,0,0,0.1)', overflow: 'hidden' }}>
      {(topic || events.length > 1) && (
        <div style={{ padding: '0.5rem 1rem', background: '#f0f0f0', fontWeight: 600, fontSize: '0.95em', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span>{topic || `Group (${events.length} events)`}</span>
          {events.length > 1 && (
            <button
              type="button"
              onClick={handleDeleteAll}
              disabled={deletingAll}
              style={{ fontSize: '0.8em', padding: '0.2rem 0.5rem', border: '1px solid #c00', background: 'white', color: '#c00', borderRadius: '4px', cursor: deletingAll ? 'wait' : 'pointer' }}
            >
              {deletingAll ? 'Deleting…' : 'Delete all'}
            </button>
          )}
        </div>
      )}
      <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
        {events.map((e) => (
          <EventItem key={e.id} event={e} onDelete={onDelete} />
        ))}
      </ul>
    </div>
  )
}

function EventItem({ event, onDelete }) {
  const [deleting, setDeleting] = useState(false)
  const desc = stripGroupIdFromDescription(event.description)

  const handleDelete = () => {
    if (deleting) return
    setDeleting(true)
    onDelete(event.id).finally(() => setDeleting(false))
  }

  return (
    <li
      style={{
        padding: '0.75rem 1rem',
        borderTop: '1px solid #eee',
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'flex-start',
      }}
    >
      <div>
        <strong>{event.summary || '(Untitled)'}</strong>
        <div style={{ fontSize: '0.9em', color: '#666', marginTop: '0.25rem' }}>
          {formatEventTime(event)}
        </div>
        {desc && (
          <div style={{ fontSize: '0.85em', color: '#888', marginTop: '0.25rem' }}>
            {desc}
          </div>
        )}
      </div>
      <button
        onClick={handleDelete}
        disabled={deleting}
        style={{
          padding: '0.25rem 0.5rem',
          fontSize: '0.85em',
          borderRadius: '4px',
          border: '1px solid #ccc',
          background: 'white',
          cursor: deleting ? 'wait' : 'pointer',
        }}
      >
        {deleting ? 'Deleting…' : 'Delete'}
      </button>
    </li>
  )
}

function formatEventTime(event) {
  const start = event.start?.dateTime || event.start?.date
  const end = event.end?.dateTime || event.end?.date
  if (!start) return ''
  if (start === end) return start
  return `${start} – ${end}`
}

export default App
