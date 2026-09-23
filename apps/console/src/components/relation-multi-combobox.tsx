import type { RelationOption } from "@/components/relation-combobox";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";

export function RelationMultiCombobox({
  values,
  options,
  onValuesChange,
  placeholder = "Select resources",
  searchPlaceholder,
}: {
  values: string[];
  options: RelationOption[];
  onValuesChange: (values: string[]) => void;
  placeholder?: string;
  searchPlaceholder?: string;
}) {
  const selected = options.filter((option) => values.includes(option.value));
  return (
    <Combobox
      items={options}
      multiple
      value={selected}
      onValueChange={(items) => onValuesChange(items.map((item) => item.value))}
      itemToStringLabel={(option) => option.label}
      itemToStringValue={(option) => option.value}
      isItemEqualToValue={(option, selectedOption) =>
        option.value === selectedOption.value
      }
    >
      <ComboboxChips className="w-full">
        <>
          {selected.map((option) => (
            <ComboboxChip key={option.value}>{option.label}</ComboboxChip>
          ))}
        </>
        <ComboboxChipsInput placeholder={searchPlaceholder ?? placeholder} />
      </ComboboxChips>
      <ComboboxContent>
        <ComboboxEmpty>No resource found.</ComboboxEmpty>
        <ComboboxList>
          {(option: RelationOption) => (
            <ComboboxItem key={option.value} value={option}>
              <span className="flex min-w-0 flex-col">
                <span className="truncate font-medium">{option.label}</span>
                {option.description && (
                  <span className="truncate text-xs text-muted-foreground">
                    {option.description}
                  </span>
                )}
              </span>
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
