import React, {useState, useCallback, useEffect, useRef} from 'react';

interface SearchBarProps {
    query: string;
    onSearch: (q: string) => void;
    onClear: () => void;
}

const DEBOUNCE_MS = 300;

const SearchBar: React.FC<SearchBarProps> = ({query, onSearch, onClear}) => {
    const [value, setValue] = useState(query);
    const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const handleChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
        const q = e.target.value;
        setValue(q);

        if (timerRef.current) {
            clearTimeout(timerRef.current);
        }

        if (q.trim()) {
            timerRef.current = setTimeout(() => {
                onSearch(q);
            }, DEBOUNCE_MS);
        } else {
            onClear();
        }
    }, [onSearch, onClear]);

    const handleClear = useCallback(() => {
        if (timerRef.current) {
            clearTimeout(timerRef.current);
        }
        setValue('');
        onClear();
    }, [onClear]);

    // Flush pending search immediately on Enter
    const handleKeyDown = useCallback((e: React.KeyboardEvent<HTMLInputElement>) => {
        if (e.key === 'Enter' && value.trim()) {
            if (timerRef.current) {
                clearTimeout(timerRef.current);
            }
            onSearch(value);
        }
    }, [value, onSearch]);

    // Cleanup on unmount
    useEffect(() => {
        return () => {
            if (timerRef.current) {
                clearTimeout(timerRef.current);
            }
        };
    }, []);

    return (
        <div className='org-directory-search-bar'>
            <input
                type='text'
                className='org-directory-search-input'
                placeholder='搜索用户/部门...'
                value={value}
                onChange={handleChange}
                onKeyDown={handleKeyDown}
                style={{
                    width: '100%',
                    padding: '8px 32px 8px 12px',
                    borderRadius: '4px',
                    border: '1px solid #ccc',
                    fontSize: '14px',
                    boxSizing: 'border-box',
                }}
            />
            {value && (
                <button
                    className='org-directory-search-clear'
                    onClick={handleClear}
                    aria-label={'清除搜索'}
                    style={{
                        position: 'absolute',
                        right: '12px',
                        top: '50%',
                        transform: 'translateY(-50%)',
                        background: 'none',
                        border: 'none',
                        cursor: 'pointer',
                        color: '#999',
                        fontSize: '16px',
                        padding: 0,
                    }}
                >
                    {'✕'}
                </button>
            )}
        </div>
    );
};

export default SearchBar;
