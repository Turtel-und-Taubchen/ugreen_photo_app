const MONTH_NAMES = [
  'Januar', 'Februar', 'März', 'April', 'Mai', 'Juni',
  'Juli', 'August', 'September', 'Oktober', 'November', 'Dezember',
];

const monthName = (mm: string): string => MONTH_NAMES[Number(mm) - 1] ?? mm;

/** `2024-07` becomes "Juli 2024". */
export function monthLabel(month: string): string {
  const [year, mm] = month.split('-');
  return `${monthName(mm)} ${year}`;
}

/** `2024-07-12` becomes "12. Juli 2024". */
export function dayLabel(day: string): string {
  const [year, mm, dd] = day.split('-');
  return `${Number(dd)}. ${monthName(mm)} ${year}`;
}
