import { Beat, Sheet } from '@/components/paper'

const stats = [
  { value: 'Floors', label: 'One per project' },
  { value: 'Teams', label: 'Tickets + ownership' },
  { value: '3 agents', label: 'Choose per conversation' },
  { value: 'Plan first', label: 'Review before build' },
]

export function StatsStrip() {
  return (
    <Sheet className="border-b border-rule">
      <div className="px-6 py-16 md:px-10 md:py-20 lg:px-14">
        <Beat index="1.3" label="Read the figures" title="What the floor is made of.">
          <table className="mt-8 w-full border-collapse text-left">
            <caption className="sr-only">Product pillars</caption>
            <thead>
              <tr className="border-y border-rule">
                <th scope="col" className="mono-label py-2 pr-4 font-normal">Item</th>
                <th scope="col" className="mono-label py-2 font-normal">Note</th>
              </tr>
            </thead>
            <tbody>
              {stats.map((s) => (
                <tr key={s.label} className="border-b border-rule">
                  <th scope="row" className="w-1/2 py-5 pr-4 align-baseline font-sans text-2xl font-medium tracking-tight text-ink md:text-3xl">
                    {s.value}
                  </th>
                  <td className="py-5 align-baseline">
                    <span className="mono-label">{s.label}</span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Beat>
      </div>
    </Sheet>
  )
}
