import { useMemo, useState } from "react";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";

type UnitOption = { value: string; label: string; description: string };
const standardUnits: UnitOption[] = [
  {
    value: "request",
    label: "Request",
    description: "One API or application request",
  },
  {
    value: "token",
    label: "Token",
    description: "One model input or output token",
  },
  {
    value: "event",
    label: "Event",
    description: "One occurrence recorded by the application",
  },
  {
    value: "transaction",
    label: "Transaction",
    description: "One completed business transaction",
  },
  {
    value: "message",
    label: "Message",
    description: "One sent or processed message",
  },
  {
    value: "email",
    label: "Email",
    description: "One sent or processed email",
  },
  {
    value: "seat",
    label: "Seat",
    description: "One active user or license seat",
  },
  { value: "byte", label: "Byte", description: "Raw data volume in bytes" },
  {
    value: "GB",
    label: "Gigabyte (GB)",
    description: "Storage or transferred data volume",
  },
  {
    value: "second",
    label: "Second",
    description: "Compute or processing duration",
  },
  {
    value: "minute",
    label: "Minute",
    description: "Compute, call, or processing duration",
  },
  {
    value: "hour",
    label: "Hour",
    description: "Compute or resource allocation duration",
  },
];

export function UnitCombobox({
  value,
  onValueChange,
}: {
  value: string;
  onValueChange: (value: string) => void;
}) {
  const [query, setQuery] = useState("");
  const items = useMemo(() => {
    const custom = query.trim();
    if (
      !custom ||
      standardUnits.some(
        (unit) =>
          unit.value.toLowerCase() === custom.toLowerCase() ||
          unit.label.toLowerCase() === custom.toLowerCase(),
      )
    )
      return standardUnits;
    return [
      ...standardUnits,
      {
        value: custom,
        label: `Use “${custom}”`,
        description: "Custom usage unit",
      },
    ];
  }, [query]);
  const selected =
    items.find((unit) => unit.value === value) ??
    (value ? { value, label: value, description: "Custom usage unit" } : null);
  return (
    <Combobox
      items={items}
      value={selected}
      onValueChange={(unit) => onValueChange(unit?.value ?? "")}
      onInputValueChange={setQuery}
      itemToStringLabel={(unit) => unit.label}
      itemToStringValue={(unit) => unit.value}
      isItemEqualToValue={(unit, selectedUnit) =>
        unit.value === selectedUnit.value
      }
    >
      <ComboboxInput
        className="w-full"
        placeholder="Search or enter a usage unit…"
        showClear={Boolean(value)}
      />
      <ComboboxContent>
        <ComboboxEmpty>No unit found.</ComboboxEmpty>
        <ComboboxList>
          {(unit: UnitOption) => (
            <ComboboxItem key={unit.value} value={unit}>
              <span className="flex min-w-0 flex-col">
                <span className="truncate font-medium">{unit.label}</span>
                <span className="truncate text-xs text-muted-foreground">
                  {unit.description}
                </span>
              </span>
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
