type NavItem = {
  id: string
  label: string
  icon: string
}

const NAV_ITEMS: NavItem[] = [
  { id: 'overview', label: 'Overview', icon: '\u25A3' },
  { id: 'population', label: 'Population', icon: '\u2630' },
  { id: 'economy', label: 'Economy', icon: '\u2261' },
  { id: 'config', label: 'Config', icon: '\u2699' },
  { id: 'history', label: 'History', icon: '\u23F1' },
]

type Props = {
  active: string | null
  onSelect: (id: string | null) => void
}

export function NavBar({ active, onSelect }: Props) {
  return (
    <nav className="navbar">
      <div className="navbar-brand">omega</div>
      <div className="navbar-items">
        {NAV_ITEMS.map(item => (
          <button
            key={item.id}
            className="navbar-item"
            data-active={active === item.id}
            onClick={() => onSelect(active === item.id ? null : item.id)}
          >
            <span className="navbar-icon">{item.icon}</span>
            <span>{item.label}</span>
          </button>
        ))}
      </div>
    </nav>
  )
}
