import { useState } from 'react'

type Props = {
  title: string
  defaultOpen?: boolean
  children: React.ReactNode
}

export function Accordion({ title, defaultOpen = false, children }: Props) {
  const [open, setOpen] = useState(defaultOpen)

  return (
    <div className="acc-section" data-open={open}>
      <button className="acc-trigger" onClick={() => setOpen(o => !o)}>
        {title}
        <span className="acc-chevron">&#9654;</span>
      </button>
      {open && <div className="acc-body">{children}</div>}
    </div>
  )
}
