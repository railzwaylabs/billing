import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";

export type RelationOption = {
  value: string;
  label: string;
  description?: string;
};

export function RelationCombobox({
  value,
  options,
  onValueChange,
  placeholder = "Select resource",
  searchPlaceholder,
  disabled = false,
}: {
  value: string;
  options: RelationOption[];
  onValueChange: (value: string) => void;
  placeholder?: string;
  searchPlaceholder?: string;
  disabled?: boolean;
}) {
  const selected = options.find((option) => option.value === value) ?? null;
  return (
    <Combobox
      items={options}
      value={selected}
      onValueChange={(option) => onValueChange(option?.value ?? "")}
      itemToStringLabel={(option) => option.label}
      itemToStringValue={(option) => option.value}
      isItemEqualToValue={(option, selectedOption) =>
        option.value === selectedOption.value
      }
      disabled={disabled}
    >
      <ComboboxInput
        className="w-full"
        placeholder={searchPlaceholder ?? placeholder}
        showClear={Boolean(value)}
      />
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
