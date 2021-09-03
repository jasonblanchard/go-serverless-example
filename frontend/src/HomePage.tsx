import React, { useEffect, useState } from 'react';
import { useRecoilState } from 'recoil';

import { idTokenState } from './state';

export default function HomePage() {
    const [meta, setMeta] = useState<Record<any, any>>();
    const [idToken] = useRecoilState(idTokenState);

    useEffect(() => {
        async function call() {
            const result = await fetch('/api/meta', {
                headers: {
                    Authorization: `Bearer ${idToken}`
                }
            });
            
            if (result.ok) {
                const body = await result.json();
                setMeta(body);
            } else {
                console.log(result.status);
            }
        }

        call();
    }, [idToken]);

    return (
        <div>
            HomePage
            <div>
                <textarea style={{ width: '100%' }} readOnly value={JSON.stringify(meta, null, 2)} rows={50} />
            </div>
        </div>
    )
}