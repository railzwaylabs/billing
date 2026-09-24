import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";

export type UnitOption = { value: string; label: string; description: string };

export function UnitCombobox({
  value,
  onValueChange,
  options,
}: {
  value: string;
  onValueChange: (value: string) => void;
  options: UnitOption[];
}) {
  const selected = options.find((unit) => unit.value === value) ?? null;
  return (
    <Combobox
      items={options}
      value={selected}
      onValueChange={(unit) => onValueChange(unit?.value ?? "")}
      itemToStringLabel={(unit) => unit.label}
      itemToStringValue={(unit) => unit.value}
      isItemEqualToValue={(unit, selectedUnit) =>
        unit.value === selectedUnit.value
      }
    >
      <ComboboxInput
        className="w-full"
        placeholder="Select a measurement unit…"
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
